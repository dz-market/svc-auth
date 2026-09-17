package user

import (
	"strings"
	"time"
	"uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    *time.Time
}

func New(email, passwordHash string, now time.Time) User {
	return User{
		ID:           uuid.NewV7(),
		Email:        NormalizeEmail(email),
		PasswordHash: passwordHash,
		CreatedAt:    now,
	}
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
