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

func (m AccessManager) Issue(userID, sessionID uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(m.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
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
