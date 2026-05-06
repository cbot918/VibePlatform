package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/yjtech/vibeplatform/internal/store"
)

var validProjectName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}$`)

type CodeServerDocker interface {
	EnsureCodeServer(ctx context.Context, userID, apiKey string) (containerID, hostPort string, err error)
	MkdirProject(ctx context.Context, containerID, projectName string) error
	Stop(ctx context.Context, containerID string) error
	ConfigureGit(ctx context.Context, containerID, gitUser, gitEmail, gitToken string) error
}

type ProjectStore interface {
	Get(userID, projectName string) (*store.ProjectInfo, error)
	List(userID string) (map[string]*store.ProjectInfo, error)
	Save(userID, projectName string, info *store.ProjectInfo) error
	Delete(userID, projectName string) error
}

type ProjectHandler struct {
	docker     CodeServerDocker
	projects   ProjectStore
	containers ContainerStore
	settings   SettingsStore
	sessions   SessionValidator
	users      UserGetter
}

func NewProjectHandler(docker CodeServerDocker, projects ProjectStore, containers ContainerStore, settings SettingsStore, sessions SessionValidator, users UserGetter) *ProjectHandler {
	return &ProjectHandler{docker: docker, projects: projects, containers: containers, settings: settings, sessions: sessions, users: users}
}

// listItem is what we return to the frontend for each project.
type listItem struct {
	Name      string `json:"name"`
	HostPort  string `json:"host_port"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func (h *ProjectHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	githubID, err := resolveGithubID(r, h.sessions, h.users)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projects, err := h.projects.List(githubID)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	cinfo, _ := h.containers.Get(githubID) // nil if no container yet

	list := make([]listItem, 0, len(projects))
	for _, p := range projects {
		item := listItem{
			Name:      p.Name,
			CreatedAt: p.CreatedAt.String(),
			Status:    "stopped",
		}
		if cinfo != nil {
			item.HostPort = cinfo.HostPort
			item.Status = cinfo.Status
		}
		list = append(list, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *ProjectHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	githubID, err := resolveGithubID(r, h.sessions, h.users)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !validProjectName.MatchString(body.Name) {
		http.Error(w, "invalid project name (lowercase alphanumeric and hyphens only)", http.StatusBadRequest)
		return
	}

	// Check API key exists
	settings, err := h.settings.Get(githubID)
	if err != nil || settings.AnthropicAPIKey == "" {
		http.Error(w, "anthropic_api_key not set — go to settings first", http.StatusBadRequest)
		return
	}

	// Prevent duplicate project name
	if _, err := h.projects.Get(githubID, body.Name); err == nil {
		http.Error(w, "project already exists", http.StatusConflict)
		return
	}

	// Ensure user's code-server container is running
	ctrID, hostPort, err := h.docker.EnsureCodeServer(r.Context(), githubID, settings.AnthropicAPIKey)
	if err != nil {
		http.Error(w, "failed to start container: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Persist container info at user level
	if err := h.containers.Save(githubID, &store.ContainerInfo{
		ContainerID: ctrID,
		HostPort:    hostPort,
		Status:      "running",
	}); err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	// Inject git credentials if configured — errors are non-fatal (git is optional)
	if settings.GitToken != "" {
		if err := h.docker.ConfigureGit(r.Context(), ctrID,
			settings.GitUser, settings.GitEmail, settings.GitToken); err != nil {
			log.Printf("warn: ConfigureGit for user %s: %v", githubID, err)
		}
	}

	// Create project folder inside container
	if err := h.docker.MkdirProject(r.Context(), ctrID, body.Name); err != nil {
		http.Error(w, "failed to create project folder: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save project record (name + timestamp only)
	info := &store.ProjectInfo{Name: body.Name}
	if err := h.projects.Save(githubID, body.Name, info); err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listItem{
		Name:     info.Name,
		HostPort: hostPort,
		Status:   "running",
	})
}

func (h *ProjectHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	githubID, err := resolveGithubID(r, h.sessions, h.users)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	name := chi.URLParam(r, "name")
	if _, err := h.projects.Get(githubID, name); err == store.ErrNotFound {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	cinfo, err := h.containers.Get(githubID)
	if err == store.ErrNotFound {
		w.WriteHeader(http.StatusOK) // already stopped
		return
	}
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	if err := h.docker.Stop(r.Context(), cinfo.ContainerID); err != nil {
		http.Error(w, "failed to stop: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cinfo.Status = "stopped"
	cinfo.ContainerID = ""
	_ = h.containers.Save(githubID, cinfo)
	w.WriteHeader(http.StatusOK)
}

func (h *ProjectHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	githubID, err := resolveGithubID(r, h.sessions, h.users)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	name := chi.URLParam(r, "name")
	if _, err := h.projects.Get(githubID, name); err == store.ErrNotFound {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	_ = h.projects.Delete(githubID, name)
	w.WriteHeader(http.StatusOK)
}

// GetProjectInfo is used by the proxy handler to resolve host port and status.
func (h *ProjectHandler) GetProjectInfo(githubID, name string) (*store.ProjectInfo, error) {
	p, err := h.projects.Get(githubID, name)
	if err != nil {
		return nil, err
	}
	cinfo, err := h.containers.Get(githubID)
	if err != nil {
		// No container — project exists but is stopped
		p.Status = "stopped"
		return p, nil
	}
	p.HostPort = cinfo.HostPort
	p.Status = cinfo.Status
	p.ContainerID = cinfo.ContainerID
	return p, nil
}
