package session

import "errors"

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrSessionNotFound      = errors.New("session not found")
)
