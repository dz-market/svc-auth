package mapper

import (
	"time"

	authv1 "github.com/dz-market/protobuf/gen/go/auth/v1"

	"github.com/dz-market/svc-auth/internal/application/auth"
)

func ToRegisterResponse(out auth.RegisterOutput, now time.Time) *authv1.RegisterResponse {
	return authv1.RegisterResponse_builder{
		Access:  ToToken(out.Access, now),
		Refresh: ToToken(out.Refresh, now),
	}.Build()
}

func ToToken(t auth.Token, now time.Time) *authv1.Token {
	return authv1.Token_builder{
		Token:     new(t.Value),
		ExpiresIn: new(expiresIn(t.ExpiresAt, now)),
	}.Build()
}

func expiresIn(expiresAt, now time.Time) int32 {
	if !expiresAt.After(now) {
		return 0
	}

	return int32(expiresAt.Sub(now).Round(time.Second) / time.Second)
}
