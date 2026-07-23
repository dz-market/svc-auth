package security

import (
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

func LoadRSAKeys(publicPEM, privatePEM []byte) (*rsa.PublicKey, *rsa.PrivateKey, error) {
	pub, err := jwt.ParseRSAPublicKeyFromPEM(publicPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse public key: %w", err)
	}

	priv, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse private key: %w", err)
	}

	return pub, priv, nil
}
