package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yjtech/vibeplatform/internal/auth"
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

	if port == "" {
		port = "3001"
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}
	redirectURL := baseURL + "/auth/github/callback"

	githubClient := auth.NewGithubClient(clientID, clientSecret, redirectURL)
	sessionManager := auth.NewSessionManager(jwtSecret, 7*24*time.Hour)
	userStore := store.NewInMemoryUserStore()

	authHandler := handler.NewAuthHandler(githubClient, sessionManager, userStore, frontendURL)

	srv := server.New(authHandler)

	log.Printf("Server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatal(err)
	}
}
