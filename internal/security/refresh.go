package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type RefreshManager struct {
	byteLen int
}

func (m RefreshManager) Generate() (raw string, hash []byte, err error) {
	b := make([]byte, m.byteLen) // nolint:makezero // sized buffer required for rand.Read
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("read random: %w", err)
	}

	raw = base64.RawURLEncoding.EncodeToString(b)

	return raw, m.Hash(raw), nil
}

func (m RefreshManager) Hash(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))

	return sum[:]
}
