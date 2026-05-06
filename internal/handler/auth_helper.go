package handler

import (
	"fmt"
	"net/http"

	"github.com/yjtech/vibeplatform/internal/model"
)

// resolveUser extracts the authenticated user from the session cookie.
func resolveUser(r *http.Request, sessions SessionValidator, users UserGetter) (*model.User, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil, err
	}
	userID, err := sessions.ValidateToken(cookie.Value)
	if err != nil {
		return nil, err
	}
	return users.GetByID(userID)
}

// resolveGithubID returns the GitHub ID string for the authenticated user.
func resolveGithubID(r *http.Request, sessions SessionValidator, users UserGetter) (string, error) {
	user, err := resolveUser(r, sessions, users)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", user.GithubID), nil
}
