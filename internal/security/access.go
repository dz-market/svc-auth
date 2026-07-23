package security

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	jwt.RegisteredClaims

	SessionID uuid.UUID `json:"sid"`
}

type AccessManager struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
	ttl        time.Duration
}

func (m AccessManager) Issue(userID, sessionID uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.ttl)
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(m.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	return signed, expiresAt, nil
}

func (m AccessManager) Parse(raw string) (Claims, error) {
	var claims Claims

	_, err := jwt.ParseWithClaims(
		raw, &claims,
		func(*jwt.Token) (any, error) {
			return m.publicKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)
	if err != nil {
		return Claims{}, err
	}

	return claims, nil
}
