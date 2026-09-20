package session

import (
	"time"
	"uuid"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func NewSession(userID uuid.UUID, ttl time.Duration, now time.Time) Session {
	return Session{
		ID:        uuid.NewV7(),
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
}

func (s Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

func (s Session) IsExpired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

func (s Session) IsActive(now time.Time) bool {
	return !s.IsRevoked() && !s.IsExpired(now)
}
