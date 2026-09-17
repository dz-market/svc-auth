package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"
	"uuid"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

var method = jwtgo.SigningMethodRS256

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
	switch {
	case opts.Key.Private == nil:
		return nil, errors.New("private key is required")
	case opts.Issuer == "":
		return nil, errors.New("issuer is required")
	case opts.Audience == nil:
		return nil, errors.New("audience is required")
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
		method, Claims{
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
