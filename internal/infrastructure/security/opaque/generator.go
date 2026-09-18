package opaque

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type Options struct {
	Length int
}

type Generator struct {
	length int
}

func New(opts Options) (*Generator, error) {
	return &Generator{
		length: opts.Length,
	}, nil
}

func (g *Generator) Generate() (value string, fingerprint []byte, err error) {
	//nolint:makezero // rand.Read fills the slice by its length; a zero-length one would read no bytes
	buf := make([]byte, g.length)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, fmt.Errorf("read random: %w", err)
	}

	value = base64.RawURLEncoding.EncodeToString(buf)

	return value, g.Fingerprint(value), nil
}

func (g *Generator) Fingerprint(value string) []byte {
	sum := sha256.Sum256([]byte(value))

	return sum[:]
}
