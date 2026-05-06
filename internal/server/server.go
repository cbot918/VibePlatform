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

func New(
	authHandler *handler.AuthHandler,
	containerHandler *handler.ContainerHandler,
	settingsHandler *handler.SettingsHandler,
	projectHandler *handler.ProjectHandler,
	proxyHandler *handler.ProxyHandler,
	debugHandler *handler.DebugHandler,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{frontendURL},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	// Auth
	r.Get("/auth/github", authHandler.HandleGithubLogin)
	r.Get("/auth/github/callback", authHandler.HandleGithubCallback)
	r.Get("/auth/me", authHandler.HandleMe)
	r.Post("/auth/logout", authHandler.HandleLogout)

	// Legacy container (ubuntu)
	r.Post("/container/start", containerHandler.HandleStart)
	r.Get("/container/status", containerHandler.HandleStatus)
	r.Post("/container/stop", containerHandler.HandleStop)

	// User settings
	r.Get("/user/settings", settingsHandler.HandleGet)
	r.Post("/user/settings", settingsHandler.HandleSave)

	// Projects (code-server)
	r.Get("/project", projectHandler.HandleList)
	r.Post("/project", projectHandler.HandleCreate)
	r.Post("/project/{name}/stop", projectHandler.HandleStop)
	r.Delete("/project/{name}", projectHandler.HandleDelete)

	// Debug / testing utilities
	r.Post("/debug/reset", debugHandler.HandleReset)

	// Proxy to code-server (must be last — wildcard)
	r.Handle("/project/{name}/*", proxyHandler)
	r.Handle("/project/{name}", proxyHandler)

	return r
}
