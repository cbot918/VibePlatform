package store

import (
	"testing"

	"github.com/yjtech/vibeplatform/internal/model"
)

func TestUpsert_NewUser(t *testing.T) {
	s := NewInMemoryUserStore()
	u, err := s.Upsert(&model.User{GithubID: 1, Login: "alice", Name: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if u.Login != "alice" {
		t.Errorf("expected login alice, got %s", u.Login)
	}
}

func TestUpsert_UpdateExisting(t *testing.T) {
	s := NewInMemoryUserStore()
	u1, _ := s.Upsert(&model.User{GithubID: 1, Login: "alice"})
	u2, err := s.Upsert(&model.User{GithubID: 1, Login: "alice-updated", Name: "Alice Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if u1.ID != u2.ID {
		t.Error("expected same ID for upsert of existing user")
	}
	if u2.Login != "alice-updated" {
		t.Errorf("expected updated login, got %s", u2.Login)
	}
}

func TestGetByGithubID(t *testing.T) {
	s := NewInMemoryUserStore()
	s.Upsert(&model.User{GithubID: 99, Login: "bob"})

	u, err := s.GetByGithubID(99)
	if err != nil {
		t.Fatal(err)
	}
	if u.Login != "bob" {
		t.Errorf("expected bob, got %s", u.Login)
	}
}

func TestGetByGithubID_NotFound(t *testing.T) {
	s := NewInMemoryUserStore()
	_, err := s.GetByGithubID(999)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByID(t *testing.T) {
	s := NewInMemoryUserStore()
	created, _ := s.Upsert(&model.User{GithubID: 7, Login: "charlie"})

	u, err := s.GetByID(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if u.Login != "charlie" {
		t.Errorf("expected charlie, got %s", u.Login)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	s := NewInMemoryUserStore()
	_, err := s.GetByID(999)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpsert_MultipleUsers_UniqueIDs(t *testing.T) {
	s := NewInMemoryUserStore()
	u1, _ := s.Upsert(&model.User{GithubID: 1, Login: "a"})
	u2, _ := s.Upsert(&model.User{GithubID: 2, Login: "b"})
	if u1.ID == u2.ID {
		t.Error("expected different IDs for different users")
	}
}
