package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type RefreshTokenGenerator struct {
	byteLen int
}

func NewRefreshTokenGenerator(byteLen int) RefreshTokenGenerator {
	return RefreshTokenGenerator{byteLen: byteLen}
}

func (g RefreshTokenGenerator) Generate() (raw string, hash []byte, err error) {
	b := make([]byte, g.byteLen) // nolint:makezero // sized buffer required for rand.Read
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("read random: %w", err)
	}

	raw = base64.RawURLEncoding.EncodeToString(b)

	return raw, HashRefreshToken(raw), nil
}

func HashRefreshToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))

	return sum[:]
}
