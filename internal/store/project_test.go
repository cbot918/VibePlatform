package store_test

import (
	"os"
	"testing"

	"github.com/yjtech/vibeplatform/internal/store"
)

func TestProjectStore_SaveAndGet(t *testing.T) {
	f, _ := os.CreateTemp("", "projects-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s, err := store.NewFileProjectStore(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	proj := &store.ProjectInfo{Name: "project1"}
	if err := s.Save("user1", "project1", proj); err != nil {
		t.Fatal(err)
	}

	got, err := s.Get("user1", "project1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "project1" {
		t.Errorf("want project1, got %q", got.Name)
	}
}

func TestProjectStore_List(t *testing.T) {
	f, _ := os.CreateTemp("", "projects-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s, _ := store.NewFileProjectStore(f.Name())
	_ = s.Save("user1", "alpha", &store.ProjectInfo{Name: "alpha"})
	_ = s.Save("user1", "beta", &store.ProjectInfo{Name: "beta"})

	projects, err := s.List("user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 2 {
		t.Errorf("want 2 projects, got %d", len(projects))
	}
}

func TestProjectStore_GetNotFound(t *testing.T) {
	f, _ := os.CreateTemp("", "projects-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s, _ := store.NewFileProjectStore(f.Name())
	_, err := s.Get("user1", "nope")
	if err != store.ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestProjectStore_Delete(t *testing.T) {
	f, _ := os.CreateTemp("", "projects-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s, _ := store.NewFileProjectStore(f.Name())
	_ = s.Save("user1", "p1", &store.ProjectInfo{Name: "p1"})
	_ = s.Delete("user1", "p1")

	_, err := s.Get("user1", "p1")
	if err != store.ErrNotFound {
		t.Errorf("want ErrNotFound after delete, got %v", err)
	}
}

func TestProjectStore_Persistence(t *testing.T) {
	f, _ := os.CreateTemp("", "projects-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s1, _ := store.NewFileProjectStore(f.Name())
	_ = s1.Save("user1", "myproject", &store.ProjectInfo{Name: "myproject"})

	s2, _ := store.NewFileProjectStore(f.Name())
	got, err := s2.Get("user1", "myproject")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "myproject" {
		t.Errorf("want myproject after reload, got %q", got.Name)
	}
}
