package argon2id

import (
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type decoded struct {
	params Params
	salt   []byte
	key    []byte
}

func encode(p Params, salt, key []byte) string {
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.MemoryKiB, p.Iterations, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

func decode(hash string) (decoded, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return decoded{}, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return decoded{}, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return decoded{}, ErrInvalidHash
	}

	if version != argon2.Version {
		return decoded{}, ErrIncompatibleVersion
	}

	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Iterations, &p.Parallelism); err != nil {
		return decoded{}, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return decoded{}, ErrInvalidHash
	}

	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return decoded{}, ErrInvalidHash
	}

	if len(salt) == 0 || len(key) == 0 || p.Iterations < 1 || p.Parallelism < 1 {
		return decoded{}, ErrInvalidHash
	}

	//nolint:gosec // decoded base64 lengths cannot overflow uint32
	p.SaltLength, p.KeyLength = uint32(len(salt)), uint32(len(key))

	return decoded{
		params: p,
		salt:   salt,
		key:    key,
	}, nil
}
