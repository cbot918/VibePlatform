package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/yjtech/vibeplatform/internal/model"
	"github.com/yjtech/vibeplatform/internal/store"
)

const (
	stateCookieName   = "oauth_state"
	sessionCookieName = "session"
)

type GithubOAuth interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	FetchUser(ctx context.Context, token *oauth2.Token) (*model.User, error)
}

type SessionService interface {
	CreateToken(userID int64) (string, error)
	ValidateToken(tokenStr string) (int64, error)
}

type AuthHandler struct {
	github      GithubOAuth
	sessions    SessionService
	users       store.UserStore
	frontendURL string
}

func NewAuthHandler(github GithubOAuth, sessions SessionService, users store.UserStore, frontendURL string) *AuthHandler {
	return &AuthHandler{github: github, sessions: sessions, users: users, frontendURL: frontendURL}
}

func (h *AuthHandler) HandleGithubLogin(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		MaxAge:   300,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		Path:     "/",
	})
	http.Redirect(w, r, h.github.AuthCodeURL(state), http.StatusFound)
}

func (h *AuthHandler) HandleGithubCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: stateCookieName, Value: "", MaxAge: -1, Path: "/", Secure: true, SameSite: http.SameSiteNoneMode})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := h.github.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "exchange failed", http.StatusInternalServerError)
		return
	}

	githubUser, err := h.github.FetchUser(r.Context(), token)
	if err != nil {
		http.Error(w, "fetch user failed", http.StatusInternalServerError)
		return
	}

	user, err := h.users.Upsert(githubUser)
	if err != nil {
		http.Error(w, "store user failed", http.StatusInternalServerError)
		return
	}

	sessionToken, err := h.sessions.CreateToken(user.ID)
	if err != nil {
		http.Error(w, "create session failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		Path:     "/",
	})
	http.Redirect(w, r, h.frontendURL, http.StatusFound)
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := h.sessions.ValidateToken(cookie.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.users.GetByID(userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/",
	})
	w.WriteHeader(http.StatusOK)
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
