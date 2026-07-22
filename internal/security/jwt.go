package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	SessionID uuid.UUID `json:"sid"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret []byte
	TTL    time.Duration
}

func NewTokenManager(secret string, TTL time.Duration) TokenManager {
	return TokenManager{
		secret: []byte(secret),
		TTL:    TTL,
	}
}

func (m TokenManager) Issue(userID, sessionID uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.TTL)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}

func (m TokenManager) Parse(raw string) (Claims, error) {
	var claims Claims

	_, err := jwt.ParseWithClaims(
		raw, claims, func(*jwt.Token) (any, error) {
			return m.secret, nil
		},
	)
	if err != nil {
		return Claims{}, err
	}

	return claims, nil
}
