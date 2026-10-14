package jwt

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
)

const MinKeyBits = 2048

type KeyPair struct {
	ID      string
	Private *rsa.PrivateKey
	Public  *rsa.PublicKey
}

func ParseKeyPair(privatePEM string) (KeyPair, error) {
	block, _ := pem.Decode([]byte(privatePEM))
	if block == nil {
		return KeyPair{}, errors.New("private key: no pem block")
	}

	private, err := parseRSAPrivate(block)
	if err != nil {
		return KeyPair{}, fmt.Errorf("private key: %w", err)
	}

	if bits := private.N.BitLen(); bits < MinKeyBits {
		return KeyPair{}, fmt.Errorf("private key: rsa key is %d bits, minimum is %d", bits, MinKeyBits)
	}

	if err := private.Validate(); err != nil {
		return KeyPair{}, fmt.Errorf("private key: validate rsa key: %w", err)
	}

	return KeyPair{
		ID:      thumbprint(&private.PublicKey),
		Private: private,
		Public:  &private.PublicKey,
	}, nil
}

func parseRSAPrivate(block *pem.Block) (*rsa.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key type %T is not rsa", parsed)
	}

	return key, nil
}

func thumbprint(pub *rsa.PublicKey) string {
	canonical := fmt.Sprintf(
		`{"e":%q,"kty":"RSA","n":%q}`,
		base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
	)

	sum := sha256.Sum256([]byte(canonical))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}
