package store

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/yjtech/vibeplatform/internal/model"
)

var ErrNotFound = errors.New("not found")

type UserStore interface {
	Upsert(u *model.User) (*model.User, error)
	GetByGithubID(githubID int64) (*model.User, error)
	GetByID(id int64) (*model.User, error)
}

type InMemoryUserStore struct {
	mu       sync.RWMutex
	byID     map[int64]*model.User
	byGithub map[int64]*model.User
	nextID   atomic.Int64
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		byID:     make(map[int64]*model.User),
		byGithub: make(map[int64]*model.User),
	}
}

func (s *InMemoryUserStore) Upsert(u *model.User) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byGithub[u.GithubID]; ok {
		existing.Login = u.Login
		existing.Name = u.Name
		existing.AvatarURL = u.AvatarURL
		existing.Email = u.Email
		return existing, nil
	}

	user := &model.User{
		ID:        s.nextID.Add(1),
		GithubID:  u.GithubID,
		Login:     u.Login,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		Email:     u.Email,
	}
	s.byID[user.ID] = user
	s.byGithub[user.GithubID] = user
	return user, nil
}

func (s *InMemoryUserStore) GetByGithubID(githubID int64) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byGithub[githubID]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *InMemoryUserStore) GetByID(id int64) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}
