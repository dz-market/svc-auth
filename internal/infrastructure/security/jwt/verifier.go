package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"uuid"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

type VerifierOptions struct {
	Key      KeyPair
	Issuer   string
	Audience []string
}

type Verifier struct {
	parser *jwtgo.Parser
	key    *rsa.PublicKey
}

func NewVerifier(opts VerifierOptions) (*Verifier, error) {
	if opts.Key.Public == nil {
		return nil, errors.New("public key is required")
	}

	return &Verifier{
		parser: jwtgo.NewParser(
			jwtgo.WithValidMethods([]string{jwtgo.SigningMethodRS256.Alg()}),
			jwtgo.WithIssuer(opts.Issuer),
			jwtgo.WithAudience(opts.Audience...),
			jwtgo.WithExpirationRequired(),
			jwtgo.WithIssuedAt(),
		),
		key: opts.Key.Public,
	}, nil
}

func (v *Verifier) Verify(token string) (userID, sessionID uuid.UUID, err error) {
	var claims Claims

	if _, err := v.parser.ParseWithClaims(token, &claims, v.keyFunc); err != nil {
		return uuid.Nil(), uuid.Nil(), fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	userID, err = uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), fmt.Errorf("%w: parse subject: %w", ErrInvalidToken, err)
	}

	if claims.SessionID == uuid.Nil() {
		return uuid.Nil(), uuid.Nil(), fmt.Errorf("%w: missing session id", ErrInvalidToken)
	}

	return userID, claims.SessionID, nil
}

func (v *Verifier) keyFunc(*jwtgo.Token) (any, error) {
	return v.key, nil
}
