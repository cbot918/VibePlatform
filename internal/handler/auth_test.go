package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"

	"github.com/yjtech/vibeplatform/internal/model"
	"github.com/yjtech/vibeplatform/internal/store"
)

// --- mocks ---

type mockGithub struct {
	authURL string
	token   *oauth2.Token
	user    *model.User
	err     error
}

func (m *mockGithub) AuthCodeURL(state string) string {
	return m.authURL + "?state=" + state
}

func (m *mockGithub) Exchange(_ context.Context, _ string) (*oauth2.Token, error) {
	return m.token, m.err
}

func (m *mockGithub) FetchUser(_ context.Context, _ *oauth2.Token) (*model.User, error) {
	return m.user, m.err
}

type mockSession struct {
	token  string
	userID int64
	err    error
}

func (m *mockSession) CreateToken(_ int64) (string, error) {
	return m.token, m.err
}

func (m *mockSession) ValidateToken(_ string) (int64, error) {
	return m.userID, m.err
}

// --- tests ---

func TestHandleGithubLogin(t *testing.T) {
	mg := &mockGithub{authURL: "https://github.com/login/oauth/authorize"}
	h := NewAuthHandler(mg, &mockSession{}, store.NewInMemoryUserStore(), "http://localhost:5173")

	req := httptest.NewRequest("GET", "/auth/github", nil)
	rr := httptest.NewRecorder()
	h.HandleGithubLogin(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rr.Code)
	}

	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header")
	}

	var stateCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == stateCookieName {
			stateCookie = c
		}
	}
	if stateCookie == nil || stateCookie.Value == "" {
		t.Fatal("expected oauth_state cookie to be set")
	}
}

func TestHandleGithubCallback_Success(t *testing.T) {
	mg := &mockGithub{
		authURL: "https://github.com/login/oauth/authorize",
		token:   &oauth2.Token{AccessToken: "gh-token"},
		user:    &model.User{GithubID: 42, Login: "testuser", Name: "Test User"},
	}
	ms := &mockSession{token: "jwt-token", userID: 1}
	userStore := store.NewInMemoryUserStore()
	h := NewAuthHandler(mg, ms, userStore, "http://localhost:5173")

	req := httptest.NewRequest("GET", "/auth/github/callback?code=abc&state=mystate", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "mystate"})

	rr := httptest.NewRecorder()
	h.HandleGithubCallback(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected 302, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Location") != "http://localhost:5173" {
		t.Errorf("unexpected redirect: %s", rr.Header().Get("Location"))
	}

	var sessionCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == sessionCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie")
	}
	if sessionCookie.Value != "jwt-token" {
		t.Errorf("expected jwt-token, got %s", sessionCookie.Value)
	}
}

func TestHandleGithubCallback_InvalidState(t *testing.T) {
	h := NewAuthHandler(&mockGithub{}, &mockSession{}, store.NewInMemoryUserStore(), "")

	req := httptest.NewRequest("GET", "/auth/github/callback?code=abc&state=wrong", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "different"})

	rr := httptest.NewRecorder()
	h.HandleGithubCallback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGithubCallback_MissingCode(t *testing.T) {
	h := NewAuthHandler(&mockGithub{}, &mockSession{}, store.NewInMemoryUserStore(), "")

	req := httptest.NewRequest("GET", "/auth/github/callback?state=mystate", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "mystate"})

	rr := httptest.NewRecorder()
	h.HandleGithubCallback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleMe_Authenticated(t *testing.T) {
	userStore := store.NewInMemoryUserStore()
	created, _ := userStore.Upsert(&model.User{GithubID: 1, Login: "testuser"})
	ms := &mockSession{userID: created.ID}
	h := NewAuthHandler(nil, ms, userStore, "")

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "valid-jwt"})

	rr := httptest.NewRecorder()
	h.HandleMe(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var u model.User
	if err := json.NewDecoder(rr.Body).Decode(&u); err != nil {
		t.Fatal(err)
	}
	if u.Login != "testuser" {
		t.Errorf("expected testuser, got %s", u.Login)
	}
}

func TestHandleMe_Unauthenticated(t *testing.T) {
	h := NewAuthHandler(nil, &mockSession{err: errors.New("no session")}, store.NewInMemoryUserStore(), "")

	req := httptest.NewRequest("GET", "/auth/me", nil)
	rr := httptest.NewRecorder()
	h.HandleMe(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleLogout(t *testing.T) {
	h := NewAuthHandler(nil, &mockSession{}, store.NewInMemoryUserStore(), "")

	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "some-token"})

	rr := httptest.NewRecorder()
	h.HandleLogout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var sessionCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == sessionCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil || sessionCookie.MaxAge != -1 {
		t.Fatal("expected session cookie to be cleared")
	}
}
