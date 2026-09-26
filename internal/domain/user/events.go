package user

import (
	"time"
	"uuid"
)

type Registered struct {
	EventID      uuid.UUID
	UserID       uuid.UUID
	RegisteredAt time.Time
}

func NewRegistered(u User) Registered {
	return Registered{
		EventID:      uuid.NewV7(),
		UserID:       u.ID,
		RegisteredAt: u.CreatedAt,
	}
}
