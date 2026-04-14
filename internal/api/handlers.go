package api

import (
	"encoding/json"
	"fmt"
	"m2apps/internal/progress"
	"m2apps/internal/storage"
	"m2apps/internal/updater"
	"net/http"
	"strings"
)

type Handler struct {
	Store    storage.Storage
	Progress *progress.Manager
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	if len(parts) < 3 || parts[0] != "apps" {
		writeErrorJSON(w, http.StatusNotFound, "endpoint not found")
		return
	}

	appID := parts[1]
	cfg, err := validateBearerToken(r, appID, h.Store)
	if err != nil {
		writeErrorJSON(w, http.StatusUnauthorized, err.Error())
		return
	}

	switch {
	case len(parts) == 4 && parts[2] == "update" && parts[3] == "check" && r.Method == http.MethodGet:
		h.handleCheckUpdate(w, appID)
	case len(parts) == 3 && parts[2] == "update" && r.Method == http.MethodPost:
		h.handleStartUpdate(w, appID)
	case len(parts) == 4 && parts[2] == "update" && parts[3] == "status" && r.Method == http.MethodGet:
		h.handleUpdateStatus(w, appID)
	case len(parts) == 3 && parts[2] == "channel" && r.Method == http.MethodPost:
		h.handleSwitchChannel(w, appID, cfg, r)
	case len(parts) == 4 && parts[2] == "auth" && parts[3] == "update" && r.Method == http.MethodPost:
		h.handleUpdateToken(w, appID, cfg, r)
	default:
		writeErrorJSON(w, http.StatusNotFound, "endpoint not found")
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
