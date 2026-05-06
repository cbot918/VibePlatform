package store

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type ProjectInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	// Runtime fields — populated from ContainerStore, not persisted to disk.
	HostPort    string `json:"-"`
	Status      string `json:"-"`
	ContainerID string `json:"-"`
}

type userProjects struct {
	Projects map[string]*ProjectInfo `json:"projects"`
}

type FileProjectStore struct {
	mu   sync.Mutex
	path string
	data map[string]*userProjects // key: userID
}

func NewFileProjectStore(path string) (*FileProjectStore, error) {
	s := &FileProjectStore{path: path, data: make(map[string]*userProjects)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileProjectStore) Get(userID, projectName string) (*ProjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data[userID]
	if !ok {
		return nil, ErrNotFound
	}
	p, ok := u.Projects[projectName]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *FileProjectStore) List(userID string) (map[string]*ProjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data[userID]
	if !ok {
		return map[string]*ProjectInfo{}, nil
	}
	// return a copy
	result := make(map[string]*ProjectInfo, len(u.Projects))
	for k, v := range u.Projects {
		result[k] = v
	}
	return result, nil
}

func (s *FileProjectStore) Save(userID, projectName string, info *ProjectInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info.CreatedAt.IsZero() {
		info.CreatedAt = time.Now()
	}
	info.Name = projectName
	if _, ok := s.data[userID]; !ok {
		s.data[userID] = &userProjects{Projects: make(map[string]*ProjectInfo)}
	}
	s.data[userID].Projects[projectName] = info
	return s.flush()
}

func (s *FileProjectStore) Delete(userID, projectName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data[userID]; ok {
		delete(u.Projects, projectName)
	}
	return s.flush()
}

func (s *FileProjectStore) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) || len(b) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.data)
}

func (s *FileProjectStore) flush() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
