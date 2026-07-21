package logic

import (
	"context"
	"errors"
	"github.com/R0n1ns/YstuPortal/internal/domain"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCorrectLogin(t *testing.T) {
	mockProvider := domain.NewMockUserProvider(t)
	mockProvider.On("GetUser", mock.Anything, "user", "pass").Return(&domain.User{}, nil)

	mockStorage := domain.NewMockUserStorage(t)
	mockStorage.On("SaveUser", mock.Anything, &domain.User{Role: "student"}).Return(nil)

	userManager, err := NewUserManager(mockProvider, mockStorage)
	require.NoError(t, err)
	require.NotNil(t, userManager)

	user, err := userManager.Login(context.Background(), "user", "pass")
	require.NoError(t, err)
	require.NotNil(t, user)
}

func TestWrongLogin(t *testing.T) {
	testCases := []struct {
		name  string
		login string
		pass  string
	}{
		{name: "noLogin", login: "", pass: "pass"},
		{name: "noPass", login: "user", pass: ""},
		{name: "noLoginPass", login: "", pass: ""},
		{name: "WrongPass", login: "user", pass: "wrong"},
	}
	for _, tc := range testCases {
		//tc:=tc
		t.Run(tc.name, func(t *testing.T) {
			mockProvider := domain.NewMockUserProvider(t)

			mockProvider.On("GetUser", mock.Anything, tc.login, tc.pass).Return((*domain.User)(nil), errors.New("invalid creds"))

			mockStorage := domain.NewMockUserStorage(t)

			userManager, err := NewUserManager(mockProvider, mockStorage)
			require.NoError(t, err)
			require.NotNil(t, userManager)

			user, err := userManager.Login(context.Background(), tc.login, tc.pass)
			require.Error(t, err)
			require.Nil(t, user)
		})

	}

}

func TestCtxErrLogin(t *testing.T) {
	mockProvider := domain.NewMockUserProvider(t)
	mockProvider.On("GetUser", mock.Anything, "user", "pass").Return(&domain.User{}, nil)

	mockStorage := domain.NewMockUserStorage(t)
	mockStorage.On("SaveUser", mock.Anything, &domain.User{Role: "student"}).Return(nil)

	userManager, err := NewUserManager(mockProvider, mockStorage)
	require.NoError(t, err)
	require.NotNil(t, userManager)

	ctx, cancel := context.WithCancel(context.Background())

	user, err := userManager.Login(ctx, "user", "pass")
	require.NoError(t, err)
	require.NotNil(t, user)

	cancel()

	user, err = userManager.Login(ctx, "user", "pass")

	require.Error(t, err)
	require.Nil(t, user)
}

func TestStorageErrLogin(t *testing.T) {
	mockProvider := domain.NewMockUserProvider(t)
	mockProvider.On("GetUser", mock.Anything, "user", "pass").Return(&domain.User{}, nil)
	mockProvider.On("GetUser", mock.Anything, "err", "err").Return(&domain.User{UserName: "err"}, nil)

	mockStorage := domain.NewMockUserStorage(t)
	mockStorage.On("SaveUser", mock.Anything, &domain.User{Role: "student"}).Return(nil)
	mockStorage.On("SaveUser", mock.Anything, &domain.User{UserName: "err", Role: "student"}).Return(errors.New("invalid storage"))

	userManager, err := NewUserManager(mockProvider, mockStorage)
	require.NoError(t, err)
	require.NotNil(t, userManager)

	user, err := userManager.Login(context.Background(), "user", "pass")
	require.NoError(t, err)
	require.NotNil(t, user)

	user, err = userManager.Login(context.Background(), "err", "err")
	require.Error(t, err)
	require.Nil(t, user)
}
