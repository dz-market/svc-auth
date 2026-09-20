package identity

import (
	"context"
	"uuid"
)

type Identity struct {
	SessionID uuid.UUID
}

type identityKey struct{}

func With(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

func From(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)

	return id, ok
}
