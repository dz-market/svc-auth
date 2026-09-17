package argon2id

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

type Params struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
	MaxInFlight int
}

type Hasher struct {
	p   Params
	sem chan struct{}
}

func New(p Params) *Hasher {
	if p.MaxInFlight < 1 {
		p.MaxInFlight = 1
	}

	return &Hasher{
		p:   p,
		sem: make(chan struct{}, p.MaxInFlight),
	}
}

func (h *Hasher) Hash(ctx context.Context, password string) (string, error) {
	//nolint:makezero // rand.Read fills the slice by its length; a zero-length one would read no bytes
	salt := make([]byte, h.p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}

	key, err := h.key(ctx, password, salt, h.p)
	if err != nil {
		return "", err
	}

	return encode(h.p, salt, key), nil
}

func (h *Hasher) key(ctx context.Context, password string, salt []byte, p Params) ([]byte, error) {
	select {
	case h.sem <- struct{}{}:
		defer func() {
			<-h.sem
		}()

	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return argon2.IDKey([]byte(password), salt, p.Iterations, p.MemoryKiB, p.Parallelism, p.KeyLength), nil
}

func encode(p Params, salt, key []byte) string {
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.MemoryKiB, p.Iterations, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}
