package auth_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/application/auth/mocks"
	"github.com/dz-market/svc-auth/internal/domain/session"
	"github.com/dz-market/svc-auth/internal/domain/user"
)

const (
	email        = "user@example.com"
	password     = "password"
	passwordHash = "password-hash"
	accessValue  = "access-token"
	refreshValue = "refresh-token"
)

const (
	accessTTL  = 15 * time.Minute
	sessionTTL = 720 * time.Hour
)

//nolint:gochecknoglobals // test fixtures
var (
	fixedNow           = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	refreshFingerprint = []byte("refresh-fingerprint")

	storedUser = user.User{
		ID:           uuid.NewV7(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    fixedNow,
	}
)

var errFailed = errors.New("failed")

type testService struct {
	t *testing.T

	service   *auth.Service
	clock     *mocks.MockClock
	users     *mocks.MockUsersRepository
	tokens    *mocks.MockRefreshTokensRepository
	sessions  *mocks.MockSessionsRepository
	hasher    *mocks.MockPasswordHasher
	issuer    *mocks.MockAccessTokenIssuer
	generator *mocks.MockRefreshTokenGenerator
	uow       *mocks.MockUnitOfWork
}

func newTestService(t *testing.T) testService {
	t.Helper()

	s := testService{
		t:         t,
		clock:     mocks.NewMockClock(t),
		users:     mocks.NewMockUsersRepository(t),
		tokens:    mocks.NewMockRefreshTokensRepository(t),
		sessions:  mocks.NewMockSessionsRepository(t),
		hasher:    mocks.NewMockPasswordHasher(t),
		issuer:    mocks.NewMockAccessTokenIssuer(t),
		generator: mocks.NewMockRefreshTokenGenerator(t),
		uow:       mocks.NewMockUnitOfWork(t),
	}

	repos := auth.Repositories{
		Users:         s.users,
		RefreshTokens: s.tokens,
		Sessions:      s.sessions,
	}

	s.service = auth.New(
		auth.Options{
			Repos:             repos,
			UoW:               s.uow,
			Hasher:            s.hasher,
			AccessTokenIssuer: s.issuer,
			RefreshGenerator:  s.generator,
			Clock:             s.clock,
			AccessTokenTTL:    accessTTL,
			SessionTTL:        sessionTTL,
			Log:               slog.New(slog.DiscardHandler),
		},
	)

	return s
}

func TestService_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(s testService)
		want    auth.RegisterOutput
		wantErr error
	}{
		{
			name: "success",
			setup: func(s testService) {
				expectClock(s)
				expectPasswordHash(s)
				expectTokensIssued(s)
				expectTransaction(s)
				expectPersistence(s)
			},
			want: auth.RegisterOutput{
				Access: auth.Token{
					Value:     accessValue,
					ExpiresAt: fixedNow.Add(accessTTL),
				},
				Refresh: auth.Token{
					Value:     refreshValue,
					ExpiresAt: fixedNow.Add(sessionTTL),
				},
			},
		},
		{
			name: "hash password error",
			setup: func(s testService) {
				expectClock(s)

				s.hasher.EXPECT().
					Hash(mock.Anything, password).
					Return("", errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "issue access token error",
			setup: func(s testService) {
				expectClock(s)
				expectPasswordHash(s)

				s.issuer.EXPECT().
					Issue(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "generate refresh token error",
			setup: func(s testService) {
				expectClock(s)
				expectPasswordHash(s)
				expectAccessIssue(s)

				s.generator.EXPECT().
					Generate().
					Return("", nil, errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "transaction error",
			setup: func(s testService) {
				expectClock(s)
				expectPasswordHash(s)
				expectTokensIssued(s)

				s.uow.EXPECT().
					Do(mock.Anything, mock.Anything).
					Return(errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "create user error",
			setup: func(s testService) {
				expectClock(s)
				expectPasswordHash(s)
				expectTokensIssued(s)
				expectTransaction(s)

				s.users.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "create session error",
			setup: func(s testService) {
				expectClock(s)
				expectPasswordHash(s)
				expectTokensIssued(s)
				expectTransaction(s)

				s.users.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(nil)

				s.sessions.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "create refresh token error",
			setup: func(s testService) {
				expectClock(s)
				expectPasswordHash(s)
				expectTokensIssued(s)
				expectTransaction(s)

				s.users.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(nil)

				s.sessions.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(nil)

				s.tokens.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(errFailed)
			},
			wantErr: errFailed,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				ts := newTestService(t)

				tt.setup(ts)

				out, err := ts.service.Register(
					t.Context(), auth.RegisterInput{
						Email:    email,
						Password: password,
					},
				)

				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
					assert.Zero(t, out)

					return
				}

				require.NoError(t, err)
				assert.Equal(t, tt.want, out)
			},
		)
	}
}

func TestService_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(s testService)
		want    auth.LoginOutput
		wantErr error
	}{
		{
			name: "success",
			setup: func(s testService) {
				expectClock(s)
				expectUserFound(s)
				expectPasswordMatch(s)
				expectAccessIssue(s)
				expectRefreshGenerate(s)
				expectTransaction(s)
				expectLoginPersistence(s)
			},
			want: auth.LoginOutput{
				Access: auth.Token{
					Value:     accessValue,
					ExpiresAt: fixedNow.Add(accessTTL),
				},
				Refresh: auth.Token{
					Value:     refreshValue,
					ExpiresAt: fixedNow.Add(sessionTTL),
				},
			},
		},
		{
			name: "user not found",
			setup: func(s testService) {
				expectClock(s)

				s.users.EXPECT().
					ByEmail(mock.Anything, email).
					Return(user.User{}, user.ErrNotFound)
			},
			wantErr: user.ErrInvalidCredentials,
		},
		{
			name: "find user error",
			setup: func(s testService) {
				expectClock(s)

				s.users.EXPECT().
					ByEmail(mock.Anything, email).
					Return(user.User{}, errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "wrong password",
			setup: func(s testService) {
				expectClock(s)
				expectUserFound(s)

				s.hasher.EXPECT().
					Verify(mock.Anything, password, passwordHash).
					Return(false, nil)
			},
			wantErr: user.ErrInvalidCredentials,
		},
		{
			name: "verify password error",
			setup: func(s testService) {
				expectClock(s)
				expectUserFound(s)

				s.hasher.EXPECT().
					Verify(mock.Anything, password, passwordHash).
					Return(false, errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "issue access token error",
			setup: func(s testService) {
				expectClock(s)
				expectUserFound(s)
				expectPasswordMatch(s)

				s.issuer.EXPECT().
					Issue(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "generate refresh token error",
			setup: func(s testService) {
				expectClock(s)
				expectUserFound(s)
				expectPasswordMatch(s)
				expectAccessIssue(s)

				s.generator.EXPECT().
					Generate().
					Return("", nil, errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "transaction error",
			setup: func(s testService) {
				expectClock(s)
				expectUserFound(s)
				expectPasswordMatch(s)
				expectAccessIssue(s)
				expectRefreshGenerate(s)

				s.uow.EXPECT().
					Do(mock.Anything, mock.Anything).
					Return(errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "create session error",
			setup: func(s testService) {
				expectClock(s)
				expectUserFound(s)
				expectPasswordMatch(s)
				expectAccessIssue(s)
				expectRefreshGenerate(s)
				expectTransaction(s)

				s.sessions.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(errFailed)
			},
			wantErr: errFailed,
		},
		{
			name: "create refresh token error",
			setup: func(s testService) {
				expectClock(s)
				expectUserFound(s)
				expectPasswordMatch(s)
				expectAccessIssue(s)
				expectRefreshGenerate(s)
				expectTransaction(s)

				s.sessions.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(nil)

				s.tokens.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(errFailed)
			},
			wantErr: errFailed,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				ts := newTestService(t)

				tt.setup(ts)

				out, err := ts.service.Login(
					t.Context(), auth.LoginInput{
						Email:    email,
						Password: password,
					},
				)

				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
					assert.Zero(t, out)

					return
				}

				require.NoError(t, err)
				assert.Equal(t, tt.want, out)
			},
		)
	}
}

func expectClock(s testService) {
	s.clock.EXPECT().
		Now().
		Return(fixedNow).
		Once()
}

func expectPasswordHash(s testService) {
	s.hasher.EXPECT().
		Hash(mock.Anything, password).
		Return(passwordHash, nil)
}

func expectUserFound(s testService) {
	s.users.EXPECT().
		ByEmail(mock.Anything, email).
		Return(storedUser, nil)
}

func expectPasswordMatch(s testService) {
	s.hasher.EXPECT().
		Verify(mock.Anything, password, passwordHash).
		Return(true, nil)
}

func expectAccessIssue(s testService) {
	s.t.Helper()

	s.issuer.EXPECT().
		Issue(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(
			func(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
				assert.NotEqual(s.t, uuid.Nil(), userID)
				assert.NotEqual(s.t, uuid.Nil(), sessionID)
				assert.Equal(s.t, fixedNow, issuedAt)
				assert.Equal(s.t, fixedNow.Add(accessTTL), expiresAt)

				return accessValue, nil
			},
		)
}

func expectRefreshGenerate(s testService) {
	s.generator.EXPECT().
		Generate().
		Return(refreshValue, refreshFingerprint, nil)
}

func expectTokensIssued(s testService) {
	s.t.Helper()

	expectAccessIssue(s)
	expectRefreshGenerate(s)
}

func expectTransaction(s testService) {
	s.uow.EXPECT().
		Do(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, fn func(auth.Repositories) error) error {
				return fn(
					auth.Repositories{
						Users:         s.users,
						RefreshTokens: s.tokens,
						Sessions:      s.sessions,
					},
				)
			},
		)
}

func expectPersistence(s testService) {
	s.t.Helper()

	var (
		userID    uuid.UUID
		sessionID uuid.UUID
	)

	s.users.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, u user.User) error {
				userID = u.ID

				assert.Equal(s.t, email, u.Email)
				assert.Equal(s.t, passwordHash, u.PasswordHash)
				assert.Equal(s.t, fixedNow, u.CreatedAt)

				return nil
			},
		)

	s.sessions.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, sess session.Session) error {
				sessionID = sess.ID

				assert.Equal(s.t, userID, sess.UserID)
				assert.Equal(s.t, fixedNow, sess.CreatedAt)
				assert.Equal(s.t, fixedNow.Add(sessionTTL), sess.ExpiresAt)

				return nil
			},
		)

	s.tokens.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, rt session.RefreshToken) error {
				assert.Equal(s.t, sessionID, rt.SessionID)
				assert.Equal(s.t, refreshFingerprint, rt.Hash)
				assert.Equal(s.t, fixedNow, rt.IssuedAt)

				return nil
			},
		)
}

func expectLoginPersistence(s testService) {
	s.t.Helper()

	var sessionID uuid.UUID

	s.sessions.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, sess session.Session) error {
				sessionID = sess.ID

				assert.Equal(s.t, storedUser.ID, sess.UserID)
				assert.Equal(s.t, fixedNow, sess.CreatedAt)
				assert.Equal(s.t, fixedNow.Add(sessionTTL), sess.ExpiresAt)

				return nil
			},
		)

	s.tokens.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, rt session.RefreshToken) error {
				assert.Equal(s.t, sessionID, rt.SessionID)
				assert.Equal(s.t, refreshFingerprint, rt.Hash)
				assert.Equal(s.t, fixedNow, rt.IssuedAt)

				return nil
			},
		)
}
