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

	hasGitToken := settings.GitToken != ""
	maskedGitToken := ""
	if hasGitToken {
		maskedGitToken = maskKey(settings.GitToken)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"has_key":          hasKey,
		"masked_key":       maskedKey,
		"git_user":         settings.GitUser,
		"git_email":        settings.GitEmail,
		"has_git_token":    hasGitToken,
		"masked_git_token": maskedGitToken,
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
		GitUser         string `json:"git_user"`
		GitEmail        string `json:"git_email"`
		GitToken        string `json:"git_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// Load existing settings and patch only non-empty fields.
	existing, err := h.store.Get(githubID)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	if body.AnthropicAPIKey != "" {
		existing.AnthropicAPIKey = body.AnthropicAPIKey
	}
	if body.GitUser != "" {
		existing.GitUser = body.GitUser
	}
	if body.GitEmail != "" {
		existing.GitEmail = body.GitEmail
	}
	if body.GitToken != "" {
		existing.GitToken = body.GitToken
	}

	if err := h.store.Save(githubID, existing); err != nil {
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
