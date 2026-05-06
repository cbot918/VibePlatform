package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yjtech/vibeplatform/internal/auth"
	dockerclient "github.com/yjtech/vibeplatform/internal/docker"
	"github.com/yjtech/vibeplatform/internal/handler"
	"github.com/yjtech/vibeplatform/internal/server"
	"github.com/yjtech/vibeplatform/internal/store"
)

func main() {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	jwtSecret := os.Getenv("JWT_SECRET")
	port := os.Getenv("PORT")
	frontendURL := os.Getenv("FRONTEND_URL")
	dataDir := os.Getenv("DATA_DIR")

	if port == "" {
		port = "3001"
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	if dataDir == "" {
		dataDir = "./data"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}
	secureCookies := os.Getenv("SECURE_COOKIES") == "true"

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	// Stores
	userStore := store.NewInMemoryUserStore()
	containerStore, err := store.NewFileContainerStore(dataDir + "/containers.json")
	if err != nil {
		log.Fatalf("init container store: %v", err)
	}
	settingsStore, err := store.NewFileSettingsStore(dataDir + "/settings.json")
	if err != nil {
		log.Fatalf("init settings store: %v", err)
	}
	projectStore, err := store.NewFileProjectStore(dataDir + "/projects.json")
	if err != nil {
		log.Fatalf("init project store: %v", err)
	}

	// Services
	githubClient := auth.NewGithubClient(clientID, clientSecret, baseURL+"/auth/github/callback")
	sessionManager := auth.NewSessionManager(jwtSecret, 7*24*time.Hour)
	docker, err := dockerclient.New()
	if err != nil {
		log.Fatalf("init docker client: %v", err)
	}

	// Handlers
	authHandler := handler.NewAuthHandler(githubClient, sessionManager, userStore, frontendURL, secureCookies)
	containerHandler := handler.NewContainerHandler(docker, containerStore, sessionManager, userStore)
	settingsHandler := handler.NewSettingsHandler(settingsStore, sessionManager, userStore)
	projectHandler := handler.NewProjectHandler(docker, projectStore, containerStore, settingsStore, sessionManager, userStore)
	proxyHandler := handler.NewProxyHandler(projectHandler, sessionManager, userStore)

	srv := server.New(authHandler, containerHandler, settingsHandler, projectHandler, proxyHandler)

	log.Printf("Server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatal(err)
	}
}
