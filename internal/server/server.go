package server

import (
	_ "embed"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yjtech/vibeplatform/internal/handler"
)

//go:embed static/index.html
var indexHTML []byte

func New(authHandler *handler.AuthHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	r.Get("/auth/github", authHandler.HandleGithubLogin)
	r.Get("/auth/github/callback", authHandler.HandleGithubCallback)
	r.Get("/auth/me", authHandler.HandleMe)
	r.Post("/auth/logout", authHandler.HandleLogout)

	return r
}
