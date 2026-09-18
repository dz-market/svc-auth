package session

import (
	"time"
	"uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	Hash      []byte
	IssuedAt  time.Time
	UsedAt    *time.Time
}

func NewRefreshToken(sessionID uuid.UUID, hash []byte, now time.Time) RefreshToken {
	return RefreshToken{
		ID:        uuid.NewV7(),
		SessionID: sessionID,
		Hash:      hash,
		IssuedAt:  now,
	}
}
