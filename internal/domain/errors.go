package domain

import "errors"

var (
	ErrEmailTaken           = errors.New("email already taken")
	ErrUserNotFound         = errors.New("user not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenReused   = errors.New("refresh token reused")
)
