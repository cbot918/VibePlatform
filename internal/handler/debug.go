package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/yjtech/vibeplatform/internal/store"
)

type DebugDocker interface {
	Stop(ctx context.Context, containerID string) error
}

type ClearableContainerStore interface {
	Get(userID string) (*store.ContainerInfo, error)
	Clear() error
}

type ClearableProjectStore interface {
	Clear() error
}

type ClearableSettingsStore interface {
	Clear() error
}

type DebugHandler struct {
	docker     DebugDocker
	containers ClearableContainerStore
	projects   ClearableProjectStore
	settings   ClearableSettingsStore
	sessions   SessionValidator
	users      UserGetter
}

func NewDebugHandler(
	docker DebugDocker,
	containers ClearableContainerStore,
	projects ClearableProjectStore,
	settings ClearableSettingsStore,
	sessions SessionValidator,
	users UserGetter,
) *DebugHandler {
	return &DebugHandler{
		docker:     docker,
		containers: containers,
		projects:   projects,
		settings:   settings,
		sessions:   sessions,
		users:      users,
	}
}

// HandleReset stops the current user's container then clears all data stores.
func (h *DebugHandler) HandleReset(w http.ResponseWriter, r *http.Request) {
	githubID, err := resolveGithubID(r, h.sessions, h.users)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Stop container if one is running
	if info, err := h.containers.Get(githubID); err == nil && info.ContainerID != "" {
		if err := h.docker.Stop(r.Context(), info.ContainerID); err != nil {
			log.Printf("warn: reset stop container %s: %v", info.ContainerID, err)
		}
	}

	// Clear all stores
	if err := h.containers.Clear(); err != nil {
		log.Printf("warn: reset clear containers: %v", err)
	}
	if err := h.projects.Clear(); err != nil {
		log.Printf("warn: reset clear projects: %v", err)
	}
	if err := h.settings.Clear(); err != nil {
		log.Printf("warn: reset clear settings: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}
