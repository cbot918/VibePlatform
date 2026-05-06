package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/yjtech/vibeplatform/internal/handler"
	"github.com/yjtech/vibeplatform/internal/store"
)

// --- in-memory project store ---

type inMemProjectStore struct {
	data map[string]map[string]*store.ProjectInfo // userID → name → info
}

func newInMemProjectStore() *inMemProjectStore {
	return &inMemProjectStore{data: make(map[string]map[string]*store.ProjectInfo)}
}
func (s *inMemProjectStore) Get(userID, name string) (*store.ProjectInfo, error) {
	if u, ok := s.data[userID]; ok {
		if p, ok := u[name]; ok {
			return p, nil
		}
	}
	return nil, store.ErrNotFound
}
func (s *inMemProjectStore) List(userID string) (map[string]*store.ProjectInfo, error) {
	if u, ok := s.data[userID]; ok {
		return u, nil
	}
	return map[string]*store.ProjectInfo{}, nil
}
func (s *inMemProjectStore) Save(userID, name string, info *store.ProjectInfo) error {
	if _, ok := s.data[userID]; !ok {
		s.data[userID] = make(map[string]*store.ProjectInfo)
	}
	s.data[userID][name] = info
	return nil
}
func (s *inMemProjectStore) Delete(userID, name string) error {
	if u, ok := s.data[userID]; ok {
		delete(u, name)
	}
	return nil
}

// --- mock docker ---

type mockCodeServerDocker struct {
	ensureFn  func(ctx context.Context, userID, apiKey string) (string, string, error)
	mkdirFn   func(ctx context.Context, containerID, projectName string) error
	stopFn    func(ctx context.Context, containerID string) error
}

func (m *mockCodeServerDocker) EnsureCodeServer(ctx context.Context, userID, apiKey string) (string, string, error) {
	if m.ensureFn != nil {
		return m.ensureFn(ctx, userID, apiKey)
	}
	return "ctr-default", "32768", nil
}
func (m *mockCodeServerDocker) MkdirProject(ctx context.Context, containerID, projectName string) error {
	if m.mkdirFn != nil {
		return m.mkdirFn(ctx, containerID, projectName)
	}
	return nil
}
func (m *mockCodeServerDocker) Stop(ctx context.Context, containerID string) error {
	if m.stopFn != nil {
		return m.stopFn(ctx, containerID)
	}
	return nil
}

// --- helpers ---

func newProjectReq(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	return r
}

func newProjectHandler(docker *mockCodeServerDocker, ps *inMemProjectStore, cs *inMemContainerStore, ss *inMemSettingsStore) *handler.ProjectHandler {
	if docker == nil {
		docker = &mockCodeServerDocker{}
	}
	if ps == nil {
		ps = newInMemProjectStore()
	}
	if cs == nil {
		cs = newInMemContainerStore()
	}
	if ss == nil {
		ss = newInMemSettingsStore()
	}
	return handler.NewProjectHandler(docker, ps, cs, ss, &mockSession{userID: 1}, &mockUserStore{githubID: 1})
}

// --- tests ---

func TestHandleListProjects_Empty(t *testing.T) {
	h := newProjectHandler(nil, nil, nil, nil)
	w := httptest.NewRecorder()
	h.HandleList(w, newProjectReq("GET", "/project"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var body []any
	json.NewDecoder(w.Body).Decode(&body)
	if len(body) != 0 {
		t.Errorf("want empty list, got %v", body)
	}
}

func TestHandleCreateProject_Success(t *testing.T) {
	ss := newInMemSettingsStore()
	_ = ss.Save("1", &store.UserSettings{AnthropicAPIKey: "sk-ant-test"})
	ps := newInMemProjectStore()
	cs := newInMemContainerStore()

	docker := &mockCodeServerDocker{
		ensureFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "ctr-abc", "32768", nil
		},
	}
	h := newProjectHandler(docker, ps, cs, ss)

	body, _ := json.Marshal(map[string]string{"name": "myproject"})
	r := httptest.NewRequest("POST", "/project", bytes.NewReader(body))
	r.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.HandleCreate(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	// Project entry saved
	got, err := ps.Get("1", "myproject")
	if err != nil {
		t.Fatalf("project not saved: %v", err)
	}
	if got.Name != "myproject" {
		t.Errorf("want name myproject, got %q", got.Name)
	}
	// Container info saved to container store
	cinfo, err := cs.Get("1")
	if err != nil {
		t.Fatalf("container not saved: %v", err)
	}
	if cinfo.ContainerID != "ctr-abc" {
		t.Errorf("want ctr-abc, got %q", cinfo.ContainerID)
	}
}

func TestHandleCreateProject_NoAPIKey(t *testing.T) {
	h := newProjectHandler(nil, nil, nil, newInMemSettingsStore()) // no API key
	body, _ := json.Marshal(map[string]string{"name": "myproject"})
	r := httptest.NewRequest("POST", "/project", bytes.NewReader(body))
	r.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.HandleCreate(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 when no API key, got %d", w.Code)
	}
}

func TestHandleStopProject(t *testing.T) {
	ps := newInMemProjectStore()
	_ = ps.Save("1", "proj1", &store.ProjectInfo{Name: "proj1"})
	cs := newInMemContainerStore()
	_ = cs.Save("1", &store.ContainerInfo{ContainerID: "ctr-xyz", HostPort: "9000", Status: "running"})

	docker := &mockCodeServerDocker{
		stopFn: func(_ context.Context, _ string) error { return nil },
	}
	h := newProjectHandler(docker, ps, cs, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "proj1")
	r := httptest.NewRequest("POST", "/project/proj1/stop", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	w := httptest.NewRecorder()
	h.HandleStop(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	// Container store should reflect stopped status
	cinfo, _ := cs.Get("1")
	if cinfo.Status != "stopped" {
		t.Errorf("want container status stopped, got %q", cinfo.Status)
	}
}
