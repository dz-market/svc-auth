package session

import "errors"

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrSessionNotFound      = errors.New("session not found")
)
