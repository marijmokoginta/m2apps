package api

import (
	"encoding/json"
	"fmt"
	"m2apps/internal/progress"
	"m2apps/internal/storage"
	"m2apps/internal/updater"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	Store     storage.Storage
	Progress  *progress.Manager
	AccessLog func(string)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	defer h.logAccess(r, recorder.statusCode, time.Since(startedAt))

	if r.Method == http.MethodGet && strings.TrimSpace(r.URL.Path) == "/health" {
		h.handleHealth(recorder)
		return
	}
	if r.Method == http.MethodGet && strings.TrimSpace(r.URL.Path) == "/ping" {
		h.handlePing(recorder)
		return
	}

	parts := splitPath(r.URL.Path)
	if len(parts) < 3 || parts[0] != "apps" {
		writeErrorJSON(recorder, http.StatusNotFound, "endpoint not found")
		return
	}

	appID := parts[1]
	cfg, err := validateBearerToken(r, appID, h.Store)
	if err != nil {
		writeErrorJSON(recorder, http.StatusUnauthorized, err.Error())
		return
	}

	switch {
	case len(parts) == 4 && parts[2] == "update" && parts[3] == "check" && r.Method == http.MethodGet:
		h.handleCheckUpdate(recorder, appID)
	case len(parts) == 3 && parts[2] == "update" && r.Method == http.MethodPost:
		h.handleStartUpdate(recorder, appID)
	case len(parts) == 4 && parts[2] == "update" && parts[3] == "status" && r.Method == http.MethodGet:
		h.handleUpdateStatus(recorder, appID)
	case len(parts) == 3 && parts[2] == "channel" && r.Method == http.MethodPost:
		h.handleSwitchChannel(recorder, appID, cfg, r)
	case len(parts) == 4 && parts[2] == "auth" && parts[3] == "update" && r.Method == http.MethodPost:
		h.handleUpdateToken(recorder, appID, cfg, r)
	default:
		writeErrorJSON(recorder, http.StatusNotFound, "endpoint not found")
	}
}

func (h *Handler) handleCheckUpdate(w http.ResponseWriter, appID string) {
	result, err := updater.Check(appID)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleStartUpdate(w http.ResponseWriter, appID string) {
	go func() {
		if err := updater.Update(appID); err != nil {
			h.Progress.Log(appID, fmt.Sprintf("update failed: %v", err))
			h.Progress.Fail(appID)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{
		"started": true,
		"app_id":  appID,
	})
}

func (h *Handler) handleUpdateStatus(w http.ResponseWriter, appID string) {
	state, ok := h.Progress.Get(appID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"app_id":  appID,
			"status":  "idle",
			"phase":   "",
			"step":    "",
			"percent": 0,
			"logs":    []string{},
		})
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *Handler) handleSwitchChannel(w http.ResponseWriter, appID string, cfg storage.AppConfig, r *http.Request) {
	var body struct {
		Channel string `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	channel := strings.ToLower(strings.TrimSpace(body.Channel))
	if channel != "stable" && channel != "beta" && channel != "alpha" {
		writeErrorJSON(w, http.StatusBadRequest, "invalid channel, use stable|beta|alpha")
		return
	}

	cfg.Channel = channel
	if err := h.Store.Save(appID, cfg); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, fmt.Sprintf("failed to update channel: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"updated": true,
		"app_id":  appID,
		"channel": channel,
	})
}

func (h *Handler) handleUpdateToken(w http.ResponseWriter, appID string, cfg storage.AppConfig, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	token := strings.TrimSpace(body.Token)
	if token == "" {
		writeErrorJSON(w, http.StatusBadRequest, "token is required")
		return
	}

	cfg.APIToken = token
	if err := h.Store.Save(appID, cfg); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, fmt.Sprintf("failed to update token: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"updated": true,
		"app_id":  appID,
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "m2apps-local-api",
	})
}

func (h *Handler) handlePing(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{
		"pong": true,
	})
}

func (h *Handler) logAccess(r *http.Request, statusCode int, duration time.Duration) {
	if h.AccessLog == nil {
		return
	}
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method == "" {
		method = "UNKNOWN"
	}
	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = "/"
	}
	h.AccessLog(fmt.Sprintf("api %s %s -> %d (%s)", method, path, statusCode, duration.Round(time.Millisecond)))
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	raw := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
