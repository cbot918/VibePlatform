package store_test

import (
	"os"
	"testing"

	"github.com/yjtech/vibeplatform/internal/store"
)

func TestContainerStore_SaveAndGet(t *testing.T) {
	f, _ := os.CreateTemp("", "containers-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s, err := store.NewFileContainerStore(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	info := &store.ContainerInfo{
		ContainerID: "abc123",
		Status:      "running",
		HostPort:    "32768",
	}
	if err := s.Save("user1", info); err != nil {
		t.Fatal(err)
	}

	got, err := s.Get("user1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ContainerID != info.ContainerID {
		t.Errorf("want ContainerID %q, got %q", info.ContainerID, got.ContainerID)
	}
	if got.HostPort != info.HostPort {
		t.Errorf("want HostPort %q, got %q", info.HostPort, got.HostPort)
	}
}

func TestContainerStore_GetNotFound(t *testing.T) {
	f, _ := os.CreateTemp("", "containers-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s, _ := store.NewFileContainerStore(f.Name())
	_, err := s.Get("nobody")
	if err != store.ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestContainerStore_Delete(t *testing.T) {
	f, _ := os.CreateTemp("", "containers-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s, _ := store.NewFileContainerStore(f.Name())
	_ = s.Save("user1", &store.ContainerInfo{ContainerID: "abc", Status: "running", HostPort: "9000"})
	_ = s.Delete("user1")

	_, err := s.Get("user1")
	if err != store.ErrNotFound {
		t.Errorf("want ErrNotFound after delete, got %v", err)
	}
}

func TestContainerStore_Persistence(t *testing.T) {
	f, _ := os.CreateTemp("", "containers-*.json")
	f.Close()
	defer os.Remove(f.Name())

	s1, _ := store.NewFileContainerStore(f.Name())
	_ = s1.Save("user1", &store.ContainerInfo{ContainerID: "abc", Status: "running", HostPort: "9000"})

	// 重新載入，確認資料還在
	s2, _ := store.NewFileContainerStore(f.Name())
	got, err := s2.Get("user1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ContainerID != "abc" {
		t.Errorf("want ContainerID abc, got %q", got.ContainerID)
	}
}
