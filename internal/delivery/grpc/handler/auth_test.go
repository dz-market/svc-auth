package handler_test

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/dz-market/protobuf/gen/go/auth/v1"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/handler"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/handler/mocks"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/identity"
	"github.com/dz-market/svc-auth/internal/domain/session"
	"github.com/dz-market/svc-auth/internal/domain/user"
)

const (
	email           = "user@example.com"
	password        = "password"
	accessValue     = "access-token"
	refreshValue    = "refresh-token"
	oldRefreshValue = "old-refresh-token"

	accessTTL  = 15 * time.Minute
	refreshTTL = 720 * time.Hour

	accessExpiresIn  = int32(15 * 60)
	refreshExpiresIn = int32(720 * 60 * 60)
)

//nolint:gochecknoglobals // test fixtures
var (
	fixedNow  = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	sessionID = uuid.NewV7()

	errFailed = errors.New("failed")
)

type testHandler struct {
	handler *handler.Auth
	service *mocks.MockAuthService
	clock   *mocks.MockClock
}

func newTestHandler(t *testing.T) testHandler {
	t.Helper()

	h := testHandler{
		service: mocks.NewMockAuthService(t),
		clock:   mocks.NewMockClock(t),
	}

	h.handler = handler.NewAuth(
		handler.Options{
			Service: h.service,
			Clock:   h.clock,
			Log:     slog.New(slog.DiscardHandler),
		},
	)

	return h
}

func tokens() (accessToken, refreshToken auth.Token) {
	accessToken = auth.Token{
		Value:     accessValue,
		ExpiresAt: fixedNow.Add(accessTTL),
	}

	refreshToken = auth.Token{
		Value:     refreshValue,
		ExpiresAt: fixedNow.Add(refreshTTL),
	}

	return accessToken, refreshToken
}

func authenticated(t *testing.T) context.Context {
	t.Helper()

	return identity.With(
		t.Context(), identity.Identity{
			SessionID: sessionID,
		},
	)
}

func TestHandler_Register(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t)

	accessToken, refreshToken := tokens()

	h.service.EXPECT().
		Register(
			mock.Anything, auth.RegisterInput{
				Email:    email,
				Password: password,
			},
		).
		Return(
			auth.RegisterOutput{
				Access:  accessToken,
				Refresh: refreshToken,
			}, nil,
		)

	h.clock.EXPECT().
		Now().
		Return(fixedNow)

	resp, err := h.handler.Register(
		t.Context(), authv1.RegisterRequest_builder{
			Email:    new(email),
			Password: new(password),
		}.Build(),
	)
	require.NoError(t, err)

	assert.Equal(t, accessValue, resp.GetAccess().GetToken())
	assert.Equal(t, accessExpiresIn, resp.GetAccess().GetExpiresIn())

	assert.Equal(t, refreshValue, resp.GetRefresh().GetToken())
	assert.Equal(t, refreshExpiresIn, resp.GetRefresh().GetExpiresIn())
}

func TestHandler_RegisterErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		give     error
		wantCode codes.Code
		wantMsg  string
	}{
		{
			name:     "email is already taken",
			give:     user.ErrEmailTaken,
			wantCode: codes.AlreadyExists,
			wantMsg:  user.ErrEmailTaken.Error(),
		},
		{
			name:     "unknown error",
			give:     errFailed,
			wantCode: codes.Internal,
			wantMsg:  "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				h := newTestHandler(t)

				h.service.EXPECT().
					Register(
						mock.Anything, auth.RegisterInput{
							Email:    email,
							Password: password,
						},
					).
					Return(auth.RegisterOutput{}, tt.give)

				resp, err := h.handler.Register(
					t.Context(), authv1.RegisterRequest_builder{
						Email:    new(email),
						Password: new(password),
					}.Build(),
				)

				require.Error(t, err)
				assert.Nil(t, resp)

				st := status.Convert(err)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Equal(t, tt.wantMsg, st.Message())
			},
		)
	}
}

func TestHandler_Login(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t)

	accessToken, refreshToken := tokens()

	h.service.EXPECT().
		Login(
			mock.Anything, auth.LoginInput{
				Email:    email,
				Password: password,
			},
		).
		Return(
			auth.LoginOutput{
				Access:  accessToken,
				Refresh: refreshToken,
			}, nil,
		)

	h.clock.EXPECT().
		Now().
		Return(fixedNow)

	resp, err := h.handler.Login(
		t.Context(), authv1.LoginRequest_builder{
			Email:    new(email),
			Password: new(password),
		}.Build(),
	)
	require.NoError(t, err)

	assert.Equal(t, accessValue, resp.GetAccess().GetToken())
	assert.Equal(t, accessExpiresIn, resp.GetAccess().GetExpiresIn())

	assert.Equal(t, refreshValue, resp.GetRefresh().GetToken())
	assert.Equal(t, refreshExpiresIn, resp.GetRefresh().GetExpiresIn())
}

func TestHandler_LoginErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		give     error
		wantCode codes.Code
		wantMsg  string
	}{
		{
			name:     "invalid credentials",
			give:     user.ErrInvalidCredentials,
			wantCode: codes.Unauthenticated,
			wantMsg:  user.ErrInvalidCredentials.Error(),
		},
		{
			name:     "unknown error",
			give:     errFailed,
			wantCode: codes.Internal,
			wantMsg:  "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				h := newTestHandler(t)

				h.service.EXPECT().
					Login(
						mock.Anything, auth.LoginInput{
							Email:    email,
							Password: password,
						},
					).
					Return(auth.LoginOutput{}, tt.give)

				resp, err := h.handler.Login(
					t.Context(), authv1.LoginRequest_builder{
						Email:    new(email),
						Password: new(password),
					}.Build(),
				)

				require.Error(t, err)
				assert.Nil(t, resp)

				st := status.Convert(err)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Equal(t, tt.wantMsg, st.Message())
			},
		)
	}
}

func TestHandler_Refresh(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t)

	accessToken, refreshToken := tokens()

	h.service.EXPECT().
		Refresh(
			mock.Anything, auth.RefreshInput{
				RefreshToken: oldRefreshValue,
			},
		).
		Return(
			auth.RefreshOutput{
				Access:  accessToken,
				Refresh: refreshToken,
			}, nil,
		)

	h.clock.EXPECT().
		Now().
		Return(fixedNow)

	resp, err := h.handler.Refresh(
		t.Context(), authv1.RefreshRequest_builder{
			RefreshToken: new(oldRefreshValue),
		}.Build(),
	)
	require.NoError(t, err)

	assert.Equal(t, accessValue, resp.GetAccess().GetToken())
	assert.Equal(t, accessExpiresIn, resp.GetAccess().GetExpiresIn())

	assert.Equal(t, refreshValue, resp.GetRefresh().GetToken())
	assert.Equal(t, refreshExpiresIn, resp.GetRefresh().GetExpiresIn())
}

func TestHandler_RefreshErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		give     error
		wantCode codes.Code
		wantMsg  string
	}{
		{
			name:     "invalid refresh token",
			give:     session.ErrInvalidRefreshToken,
			wantCode: codes.Unauthenticated,
			wantMsg:  session.ErrInvalidRefreshToken.Error(),
		},
		{
			name:     "unknown error",
			give:     errFailed,
			wantCode: codes.Internal,
			wantMsg:  "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				h := newTestHandler(t)

				h.service.EXPECT().
					Refresh(
						mock.Anything, auth.RefreshInput{
							RefreshToken: oldRefreshValue,
						},
					).
					Return(auth.RefreshOutput{}, tt.give)

				resp, err := h.handler.Refresh(
					t.Context(), authv1.RefreshRequest_builder{
						RefreshToken: new(oldRefreshValue),
					}.Build(),
				)

				require.Error(t, err)
				assert.Nil(t, resp)

				st := status.Convert(err)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Equal(t, tt.wantMsg, st.Message())
			},
		)
	}
}

func TestHandler_Logout(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t)

	h.service.EXPECT().
		Logout(
			mock.Anything, auth.LogoutInput{
				SessionID: sessionID,
			},
		).
		Return(nil)

	resp, err := h.handler.Logout(authenticated(t), authv1.LogoutRequest_builder{}.Build())
	require.NoError(t, err)

	assert.NotNil(t, resp)
}

func TestHandler_LogoutWithoutIdentity(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t)

	resp, err := h.handler.Logout(t.Context(), authv1.LogoutRequest_builder{}.Build())

	require.Error(t, err)
	assert.Nil(t, resp)

	st := status.Convert(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "invalid access token", st.Message())
}

func TestHandler_LogoutError(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t)

	h.service.EXPECT().
		Logout(
			mock.Anything, auth.LogoutInput{
				SessionID: sessionID,
			},
		).
		Return(errFailed)

	resp, err := h.handler.Logout(authenticated(t), authv1.LogoutRequest_builder{}.Build())

	require.Error(t, err)
	assert.Nil(t, resp)

	st := status.Convert(err)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal error", st.Message())
}
