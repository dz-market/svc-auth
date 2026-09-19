package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"
	"uuid"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwtgo.RegisteredClaims

	SessionID uuid.UUID `json:"sid"`
}

type IssuerOptions struct {
	Key      KeyPair
	Issuer   string
	Audience []string
}

type Issuer struct {
	key      *rsa.PrivateKey
	kid      string
	issuer   string
	audience []string
}

func NewIssuer(opts IssuerOptions) (*Issuer, error) {
	if opts.Key.Private == nil {
		return nil, errors.New("private key is required")
	}

	return &Issuer{
		key:      opts.Key.Private,
		kid:      opts.Key.ID,
		issuer:   opts.Issuer,
		audience: opts.Audience,
	}, nil
}

func (i *Issuer) Issue(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
	token := jwtgo.NewWithClaims(
		jwtgo.SigningMethodRS256, Claims{
			SessionID: sessionID,
			RegisteredClaims: jwtgo.RegisteredClaims{
				Issuer:    i.issuer,
				Subject:   userID.String(),
				Audience:  i.audience,
				ID:        uuid.NewV7().String(),
				IssuedAt:  jwtgo.NewNumericDate(issuedAt),
				NotBefore: jwtgo.NewNumericDate(issuedAt),
				ExpiresAt: jwtgo.NewNumericDate(expiresAt),
			},
		},
	)

	token.Header["kid"] = i.kid

	signed, err := token.SignedString(i.key)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}
