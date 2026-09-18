package argon2id

import "errors"

var (
	ErrInvalidHash         = errors.New("invalid hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)
