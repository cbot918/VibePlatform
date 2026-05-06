package auth

import (
	"testing"
	"time"
)

func TestCreateToken(t *testing.T) {
	sm := NewSessionManager("test-secret-32-chars-long-enough", 1*time.Hour)
	token, err := sm.CreateToken(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestValidateToken(t *testing.T) {
	sm := NewSessionManager("test-secret-32-chars-long-enough", 1*time.Hour)
	token, _ := sm.CreateToken(42)

	userID, err := sm.ValidateToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != 42 {
		t.Errorf("expected userID 42, got %d", userID)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	sm := NewSessionManager("test-secret-32-chars-long-enough", -1*time.Second)
	token, _ := sm.CreateToken(1)

	_, err := sm.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	sm := NewSessionManager("test-secret-32-chars-long-enough", 1*time.Hour)

	_, err := sm.ValidateToken("not.a.valid.token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	sm1 := NewSessionManager("secret-one-32-chars-long-enough!", 1*time.Hour)
	sm2 := NewSessionManager("secret-two-32-chars-long-enough!", 1*time.Hour)
	token, _ := sm1.CreateToken(1)

	_, err := sm2.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}
