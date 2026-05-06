package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yjtech/vibeplatform/internal/handler"
	"github.com/yjtech/vibeplatform/internal/store"
)

func TestIndexPage_Returns200(t *testing.T) {
	srv := New(
		handler.NewAuthHandler(nil, nil, store.NewInMemoryUserStore(), ""),
		handler.NewContainerHandler(nil, nil, nil, nil),
		handler.NewSettingsHandler(nil, nil, nil),
		handler.NewProjectHandler(nil, nil, nil, nil, nil, nil),
		handler.NewProxyHandler(nil, nil, nil),
	)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestIndexPage_ContentType(t *testing.T) {
	srv := New(
		handler.NewAuthHandler(nil, nil, store.NewInMemoryUserStore(), ""),
		handler.NewContainerHandler(nil, nil, nil, nil),
		handler.NewSettingsHandler(nil, nil, nil),
		handler.NewProjectHandler(nil, nil, nil, nil, nil, nil),
		handler.NewProxyHandler(nil, nil, nil),
	)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html, got %s", ct)
	}
}

func TestIndexPage_ContainsGithubLink(t *testing.T) {
	srv := New(
		handler.NewAuthHandler(nil, nil, store.NewInMemoryUserStore(), ""),
		handler.NewContainerHandler(nil, nil, nil, nil),
		handler.NewSettingsHandler(nil, nil, nil),
		handler.NewProjectHandler(nil, nil, nil, nil, nil, nil),
		handler.NewProxyHandler(nil, nil, nil),
	)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if !strings.Contains(rr.Body.String(), "/auth/github") {
		t.Error("expected page to contain /auth/github link")
	}
}
