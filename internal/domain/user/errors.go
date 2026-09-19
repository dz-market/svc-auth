package user

import "errors"

var (
	ErrEmailTaken         = errors.New("email is already taken")
	ErrNotFound           = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
