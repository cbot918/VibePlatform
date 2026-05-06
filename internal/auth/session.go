package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type SessionManager struct {
	secret []byte
	ttl    time.Duration
}

func NewSessionManager(secret string, ttl time.Duration) *SessionManager {
	return &SessionManager{secret: []byte(secret), ttl: ttl}
}

type sessionClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func (s *SessionManager) CreateToken(userID int64) (string, error) {
	c := sessionClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
}

func (s *SessionManager) ValidateToken(tokenStr string) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &sessionClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return 0, err
	}
	c, ok := token.Claims.(*sessionClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token")
	}
	return c.UserID, nil
}
