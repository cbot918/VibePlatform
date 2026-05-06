package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yjtech/vibeplatform/internal/handler"
	"github.com/yjtech/vibeplatform/internal/store"
)

type inMemSettingsStore struct {
	data map[string]*store.UserSettings
}

func newInMemSettingsStore() *inMemSettingsStore {
	return &inMemSettingsStore{data: make(map[string]*store.UserSettings)}
}
func (s *inMemSettingsStore) Get(userID string) (*store.UserSettings, error) {
	if v, ok := s.data[userID]; ok {
		return v, nil
	}
	return &store.UserSettings{}, nil
}
func (s *inMemSettingsStore) Save(userID string, settings *store.UserSettings) error {
	s.data[userID] = settings
	return nil
}

func TestHandleGetSettings_Empty(t *testing.T) {
	h := handler.NewSettingsHandler(newInMemSettingsStore(), &mockSession{userID: 1}, &mockUserStore{githubID: 1})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/user/settings", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	h.HandleGet(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	// key 為空時應有 has_key: false
	if body["has_key"] != false {
		t.Errorf("want has_key false, got %v", body["has_key"])
	}
}

func TestHandlePostSettings_SavesKey(t *testing.T) {
	ss := newInMemSettingsStore()
	h := handler.NewSettingsHandler(ss, &mockSession{userID: 1}, &mockUserStore{githubID: 1})
	body, _ := json.Marshal(map[string]string{"anthropic_api_key": "sk-ant-test"})
	r := httptest.NewRequest("POST", "/user/settings", bytes.NewReader(body))
	r.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.HandleSave(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	saved, _ := ss.Get("1")
	if saved.AnthropicAPIKey != "sk-ant-test" {
		t.Errorf("want sk-ant-test saved, got %q", saved.AnthropicAPIKey)
	}
}

func TestHandleGetSettings_MasksKey(t *testing.T) {
	ss := newInMemSettingsStore()
	_ = ss.Save("1", &store.UserSettings{AnthropicAPIKey: "sk-ant-abcdefghij"})
	h := handler.NewSettingsHandler(ss, &mockSession{userID: 1}, &mockUserStore{githubID: 1})
	r := httptest.NewRequest("GET", "/user/settings", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	w := httptest.NewRecorder()
	h.HandleGet(w, r)
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["has_key"] != true {
		t.Errorf("want has_key true, got %v", body["has_key"])
	}
	// masked key 不應該暴露完整 key
	maskedKey, _ := body["masked_key"].(string)
	if maskedKey == "sk-ant-abcdefghij" {
		t.Error("masked_key should not return full key")
	}
}
