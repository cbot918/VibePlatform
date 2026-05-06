package store_test

import (
	"os"
	"testing"

	"github.com/yjtech/vibeplatform/internal/store"
)

func TestSettingsStore_SaveAndGet(t *testing.T) {
	f, _ := os.CreateTemp("", "settings-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s, err := store.NewFileSettingsStore(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Save("user1", &store.UserSettings{AnthropicAPIKey: "sk-ant-abc"}); err != nil {
		t.Fatal(err)
	}

	got, err := s.Get("user1")
	if err != nil {
		t.Fatal(err)
	}
	if got.AnthropicAPIKey != "sk-ant-abc" {
		t.Errorf("want sk-ant-abc, got %q", got.AnthropicAPIKey)
	}
}

func TestSettingsStore_GetNotFound_ReturnsEmpty(t *testing.T) {
	f, _ := os.CreateTemp("", "settings-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s, _ := store.NewFileSettingsStore(f.Name())
	got, err := s.Get("nobody")
	if err != nil {
		t.Fatal(err)
	}
	if got.AnthropicAPIKey != "" {
		t.Errorf("want empty key, got %q", got.AnthropicAPIKey)
	}
}

func TestSettingsStore_Persistence(t *testing.T) {
	f, _ := os.CreateTemp("", "settings-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s1, _ := store.NewFileSettingsStore(f.Name())
	_ = s1.Save("user1", &store.UserSettings{AnthropicAPIKey: "sk-ant-xyz"})

	s2, _ := store.NewFileSettingsStore(f.Name())
	got, err := s2.Get("user1")
	if err != nil {
		t.Fatal(err)
	}
	if got.AnthropicAPIKey != "sk-ant-xyz" {
		t.Errorf("want sk-ant-xyz after reload, got %q", got.AnthropicAPIKey)
	}
}
