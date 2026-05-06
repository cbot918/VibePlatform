package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yjtech/vibeplatform/internal/handler"
	"github.com/yjtech/vibeplatform/internal/model"
	"github.com/yjtech/vibeplatform/internal/store"
)

// --- mock Docker ---
type mockDocker struct {
	startFn  func(ctx context.Context, userID string) (string, string, error)
	stopFn   func(ctx context.Context, containerID string) error
	statusFn func(ctx context.Context, containerID string) (string, error)
}

func (m *mockDocker) Start(ctx context.Context, userID string) (string, string, error) {
	return m.startFn(ctx, userID)
}
func (m *mockDocker) Stop(ctx context.Context, containerID string) error {
	return m.stopFn(ctx, containerID)
}
func (m *mockDocker) Status(ctx context.Context, containerID string) (string, error) {
	return m.statusFn(ctx, containerID)
}

// --- mock Session ---
type mockSession struct{ userID int64 }

func (m *mockSession) ValidateToken(token string) (int64, error) { return m.userID, nil }

// --- mock User store ---
type mockUserStore struct{ githubID int64 }

func (m *mockUserStore) GetByID(id int64) (*model.User, error) {
	return &model.User{ID: id, GithubID: m.githubID}, nil
}

// --- helpers ---
func newContainerReq(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	return r
}

func TestHandleContainerStart_CreatesContainer(t *testing.T) {
	cs := newInMemContainerStore()
	docker := &mockDocker{
		startFn: func(_ context.Context, _ string) (string, string, error) {
			return "ctr-abc", "32768", nil
		},
	}

	h := handler.NewContainerHandler(docker, cs, &mockSession{userID: 1}, &mockUserStore{githubID: 1})
	w := httptest.NewRecorder()
	h.HandleStart(w, newContainerReq("POST", "/container/start"))

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["container_id"] != "ctr-abc" {
		t.Errorf("want container_id ctr-abc, got %v", body["container_id"])
	}
}

func TestHandleContainerStatus_NotFound(t *testing.T) {
	cs := newInMemContainerStore()
	h := handler.NewContainerHandler(&mockDocker{}, cs, &mockSession{userID: 1}, &mockUserStore{githubID: 1})
	w := httptest.NewRecorder()
	h.HandleStatus(w, newContainerReq("GET", "/container/status"))
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestHandleContainerStop_RemovesEntry(t *testing.T) {
	cs := newInMemContainerStore()
	_ = cs.Save("1", &store.ContainerInfo{ContainerID: "ctr-abc", Status: "running", HostPort: "9000"})

	docker := &mockDocker{
		stopFn: func(_ context.Context, _ string) error { return nil },
	}
	h := handler.NewContainerHandler(docker, cs, &mockSession{userID: 1}, &mockUserStore{githubID: 1})
	w := httptest.NewRecorder()
	h.HandleStop(w, newContainerReq("POST", "/container/stop"))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	_, err := cs.Get("1")
	if err != store.ErrNotFound {
		t.Error("want entry deleted after stop")
	}
}

// --- in-memory container store for tests ---
type inMemContainerStore struct {
	data map[string]*store.ContainerInfo
}

func newInMemContainerStore() *inMemContainerStore {
	return &inMemContainerStore{data: make(map[string]*store.ContainerInfo)}
}
func (s *inMemContainerStore) Get(userID string) (*store.ContainerInfo, error) {
	info, ok := s.data[userID]
	if !ok {
		return nil, store.ErrNotFound
	}
	return info, nil
}
func (s *inMemContainerStore) Save(userID string, info *store.ContainerInfo) error {
	s.data[userID] = info
	return nil
}
func (s *inMemContainerStore) Delete(userID string) error {
	delete(s.data, userID)
	return nil
}
