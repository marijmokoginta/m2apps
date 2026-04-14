package api

import (
	"encoding/json"
	"fmt"
	"m2apps/internal/storage"
	"net/http"
	"strings"
)

func validateBearerToken(r *http.Request, appID string, store storage.Storage) (storage.AppConfig, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return storage.AppConfig{}, fmt.Errorf("missing authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return storage.AppConfig{}, fmt.Errorf("invalid authorization format")
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if token == "" {
		return storage.AppConfig{}, fmt.Errorf("empty bearer token")
	}

	cfg, err := store.Load(appID)
	if err != nil {
		return storage.AppConfig{}, fmt.Errorf("failed to load app metadata: %w", err)
	}

	expected := strings.TrimSpace(cfg.APIToken)
	if expected == "" {
		expected = strings.TrimSpace(cfg.Token)
	}

	if expected == "" || token != expected {
		return storage.AppConfig{}, fmt.Errorf("invalid token")
	}

	return cfg, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErrorJSON(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":   true,
		"message": message,
	})
}
