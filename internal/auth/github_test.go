package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestAuthCodeURL_ContainsState(t *testing.T) {
	g := NewGithubClient("client-id", "client-secret", "http://localhost:8080/callback")
	url := g.AuthCodeURL("my-state")
	if !strings.Contains(url, "state=my-state") {
		t.Errorf("expected state in URL, got: %s", url)
	}
	if !strings.Contains(url, "github.com") {
		t.Errorf("expected github.com in URL, got: %s", url)
	}
}

func TestFetchUser_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         float64(12345),
			"login":      "testuser",
			"name":       "Test User",
			"avatar_url": "https://avatars.github.com/u/12345",
			"email":      "test@example.com",
		})
	}))
	defer srv.Close()

	g := NewGithubClient("id", "secret", "http://localhost/cb")
	g.apiBase = srv.URL

	token := &oauth2.Token{AccessToken: "fake-token"}
	user, err := g.FetchUser(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if user.GithubID != 12345 {
		t.Errorf("expected GithubID 12345, got %d", user.GithubID)
	}
	if user.Login != "testuser" {
		t.Errorf("expected login testuser, got %s", user.Login)
	}
}

func TestFetchUser_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	g := NewGithubClient("id", "secret", "http://localhost/cb")
	g.apiBase = srv.URL

	_, err := g.FetchUser(context.Background(), &oauth2.Token{AccessToken: "bad-token"})
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}
