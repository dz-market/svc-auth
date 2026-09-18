package jwt

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
)

const MinKeyBits = 2048

type KeyPair struct {
	ID      string
	Private *rsa.PrivateKey
	Public  *rsa.PublicKey
}

func LoadKeyPair(privatePath, publicPath string) (KeyPair, error) {
	private, err := loadPrivate(privatePath)
	if err != nil {
		return KeyPair{}, err
	}

	public, err := loadPublic(publicPath)
	if err != nil {
		return KeyPair{}, err
	}

	if !private.PublicKey.Equal(public) {
		return KeyPair{}, fmt.Errorf("key mismatch: %s is not the public half of %s", publicPath, privatePath)
	}

	return KeyPair{
		ID:      thumbprint(public),
		Private: private,
		Public:  public,
	}, nil
}

func loadPrivate(path string) (*rsa.PrivateKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}

	key, err := parseRSAPrivate(block)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	if bits := key.N.BitLen(); bits < MinKeyBits {
		return nil, fmt.Errorf("%s: rsa key is %d bits, minimum is %d", path, bits, MinKeyBits)
	}

	if err := key.Validate(); err != nil {
		return nil, fmt.Errorf("%s: validate rsa key: %w", path, err)
	}

	return key, nil
}

func loadPublic(path string) (*rsa.PublicKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%s: parse public key: %w", path, err)
	}

	pub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%s: key type %T is not rsa", path, parsed)
	}

	return pub, nil
}

func readPEM(path string) (*pem.Block, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}

	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("%s: no pem block", path)
	}

	return block, nil
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
