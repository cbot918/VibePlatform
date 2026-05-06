package store

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type ContainerInfo struct {
	ContainerID string    `json:"container_id"`
	Status      string    `json:"status"`
	HostPort    string    `json:"host_port"`
	CreatedAt   time.Time `json:"created_at"`
}

type FileContainerStore struct {
	mu   sync.Mutex
	path string
	data map[string]*ContainerInfo
}

func NewFileContainerStore(path string) (*FileContainerStore, error) {
	s := &FileContainerStore{path: path, data: make(map[string]*ContainerInfo)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileContainerStore) Get(userID string) (*ContainerInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.data[userID]
	if !ok {
		return nil, ErrNotFound
	}
	return info, nil
}

func (s *FileContainerStore) Save(userID string, info *ContainerInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info.CreatedAt.IsZero() {
		info.CreatedAt = time.Now()
	}
	s.data[userID] = info
	return s.flush()
}

func (s *FileContainerStore) Delete(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, userID)
	return s.flush()
}

func (s *FileContainerStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]*ContainerInfo)
	return s.flush()
}

func (s *FileContainerStore) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) || len(b) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.data)
}

func (s *FileContainerStore) flush() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
