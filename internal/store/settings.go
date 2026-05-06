package store

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type UserSettings struct {
	AnthropicAPIKey string `json:"anthropic_api_key"`
	GitUser         string `json:"git_user"`
	GitEmail        string `json:"git_email"`
	GitToken        string `json:"git_token"`
}

type FileSettingsStore struct {
	mu   sync.Mutex
	path string
	data map[string]*UserSettings
}

func NewFileSettingsStore(path string) (*FileSettingsStore, error) {
	s := &FileSettingsStore{path: path, data: make(map[string]*UserSettings)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Get returns settings for the user. Returns empty settings (not error) if not found.
func (s *FileSettingsStore) Get(userID string) (*UserSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if settings, ok := s.data[userID]; ok {
		return settings, nil
	}
	return &UserSettings{}, nil
}

func (s *FileSettingsStore) Save(userID string, settings *UserSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[userID] = settings
	return s.flush()
}

func (s *FileSettingsStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]*UserSettings)
	return s.flush()
}

func (s *FileSettingsStore) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) || len(b) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.data)
}

func (s *FileSettingsStore) flush() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
