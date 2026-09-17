package argon2id

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
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

func New(p Params) (*Hasher, error) {
	switch {
	case p.MemoryKiB < 15*1024:
		return nil, fmt.Errorf("memory %d KiB is below the 15 MiB minimum", p.MemoryKiB)
	case p.Iterations < 1:
		return nil, errors.New("iterations must be >= 1")
	case p.Parallelism < 1:
		return nil, errors.New("parallelism must be >= 1")
	case p.SaltLength < 16:
		return nil, errors.New("saltLength must be >= 16")
	case p.KeyLength < 32:
		return nil, errors.New("keyLength must be >= 32")
	}

	return &Hasher{
		p:   p,
		sem: make(chan struct{}, p.MaxInFlight),
	}, nil
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
