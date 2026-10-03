package mapper

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"time"

	authv1 "github.com/dz-market/protobuf/gen/go/auth/api/v1"

	"github.com/dz-market/svc-auth/internal/application/auth"
)

func ToRegisterResponse(out auth.RegisterOutput, now time.Time) *authv1.RegisterResponse {
	return authv1.RegisterResponse_builder{
		Access:  ToToken(out.Access, now),
		Refresh: ToToken(out.Refresh, now),
	}.Build()
}

func ToLoginResponse(out auth.LoginOutput, now time.Time) *authv1.LoginResponse {
	return authv1.LoginResponse_builder{
		Access:  ToToken(out.Access, now),
		Refresh: ToToken(out.Refresh, now),
	}.Build()
}

func ToRefreshResponse(out auth.RefreshOutput, now time.Time) *authv1.RefreshResponse {
	return authv1.RefreshResponse_builder{
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

func ToJwk(id, alg string, key *rsa.PublicKey) *authv1.Jwk {
	return authv1.Jwk_builder{
		Kty: new("RSA"),
		Use: new("sig"),
		Kid: new(id),
		Alg: new(alg),
		N:   new(base64.RawURLEncoding.EncodeToString(key.N.Bytes())),
		E:   new(base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())),
	}.Build()
}

func expiresIn(expiresAt, now time.Time) int32 {
	if !expiresAt.After(now) {
		return 0
	}

	//nolint:gosec // bounded by the configured TTLs
	return int32(expiresAt.Sub(now).Round(time.Second) / time.Second)
}
