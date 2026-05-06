package server

import (
	_ "embed"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/yjtech/vibeplatform/internal/handler"
)

//go:embed static/index.html
var indexHTML []byte

func New(authHandler *handler.AuthHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{frontendURL},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

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
