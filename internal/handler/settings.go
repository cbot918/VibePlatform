package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/yjtech/vibeplatform/internal/store"
)

type SettingsStore interface {
	Get(userID string) (*store.UserSettings, error)
	Save(userID string, settings *store.UserSettings) error
}

type SettingsHandler struct {
	store    SettingsStore
	sessions SessionValidator
	users    UserGetter
}

func NewSettingsHandler(store SettingsStore, sessions SessionValidator, users UserGetter) *SettingsHandler {
	return &SettingsHandler{store: store, sessions: sessions, users: users}
}

func (h *SettingsHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	githubID, err := resolveGithubID(r, h.sessions, h.users)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	settings, err := h.store.Get(githubID)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	hasKey := settings.AnthropicAPIKey != ""
	maskedKey := ""
	if hasKey {
		maskedKey = maskKey(settings.AnthropicAPIKey)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"has_key":    hasKey,
		"masked_key": maskedKey,
	})
}

func (h *SettingsHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	githubID, err := resolveGithubID(r, h.sessions, h.users)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		AnthropicAPIKey string `json:"anthropic_api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.AnthropicAPIKey == "" {
		http.Error(w, "anthropic_api_key required", http.StatusBadRequest)
		return
	}

	if err := h.store.Save(githubID, &store.UserSettings{AnthropicAPIKey: body.AnthropicAPIKey}); err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// maskKey shows first 7 chars and last 4 chars, masking the middle.
func maskKey(key string) string {
	if len(key) <= 11 {
		return strings.Repeat("*", len(key))
	}
	return key[:7] + strings.Repeat("*", len(key)-11) + key[len(key)-4:]
}
