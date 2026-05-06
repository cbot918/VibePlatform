package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yjtech/vibeplatform/internal/model"
	"github.com/yjtech/vibeplatform/internal/store"
)

type DockerClient interface {
	Start(ctx context.Context, userID string) (containerID, hostPort string, err error)
	Stop(ctx context.Context, containerID string) error
	Status(ctx context.Context, containerID string) (string, error)
}

type ContainerStore interface {
	Get(userID string) (*store.ContainerInfo, error)
	Save(userID string, info *store.ContainerInfo) error
	Delete(userID string) error
}

type SessionValidator interface {
	ValidateToken(token string) (int64, error)
}

type UserGetter interface {
	GetByID(id int64) (*model.User, error)
}

type ContainerHandler struct {
	docker   DockerClient
	store    ContainerStore
	sessions SessionValidator
	users    UserGetter
}

func NewContainerHandler(docker DockerClient, store ContainerStore, sessions SessionValidator, users UserGetter) *ContainerHandler {
	return &ContainerHandler{docker: docker, store: store, sessions: sessions, users: users}
}

func (h *ContainerHandler) githubID(r *http.Request) (string, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", err
	}
	userID, err := h.sessions.ValidateToken(cookie.Value)
	if err != nil {
		return "", err
	}
	user, err := h.users.GetByID(userID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", user.GithubID), nil
}

func (h *ContainerHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	githubID, err := h.githubID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 已有 container 則直接回傳
	if info, err := h.store.Get(githubID); err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
		return
	}

	ctrID, hostPort, err := h.docker.Start(r.Context(), githubID)
	if err != nil {
		http.Error(w, "failed to start container: "+err.Error(), http.StatusInternalServerError)
		return
	}

	info := &store.ContainerInfo{
		ContainerID: ctrID,
		Status:      "running",
		HostPort:    hostPort,
	}
	if err := h.store.Save(githubID, info); err != nil {
		http.Error(w, "failed to save container info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *ContainerHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	githubID, err := h.githubID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	info, err := h.store.Get(githubID)
	if err == store.ErrNotFound {
		http.Error(w, "no container", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	status, err := h.docker.Status(r.Context(), info.ContainerID)
	if err != nil {
		http.Error(w, "docker error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	info.Status = status
	_ = h.store.Save(githubID, info)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *ContainerHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	githubID, err := h.githubID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	info, err := h.store.Get(githubID)
	if err == store.ErrNotFound {
		http.Error(w, "no container", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	if err := h.docker.Stop(r.Context(), info.ContainerID); err != nil {
		http.Error(w, "failed to stop: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = h.store.Delete(githubID)
	w.WriteHeader(http.StatusOK)
}
