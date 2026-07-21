package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Email     Email
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewUser(email, password string) User {
	return User{
		ID:       uuid.New(),
		Email:    NewEmail(email),
		Password: password,
	}
}

type Email string

func NewEmail(raw string) Email {
	raw = strings.TrimSpace(raw)
	raw = strings.ToLower(raw)

	return Email(raw)
}

func (e Email) String() string {
	return string(e)
}
