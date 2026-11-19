package user

import (
	"time"
	"uuid"
)

type User struct {
	ID           uuid.UUID
	Email        Email
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    *time.Time
}

func New(email Email, passwordHash string, now time.Time) User {
	return User{
		ID:           uuid.NewV7(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
	}
}
