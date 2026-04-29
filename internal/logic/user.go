package logic

import (
	"YstuPortal/internal/domain"
	"context"

	"github.com/google/uuid"
)

type UserManager struct {
	UserProvider domain.UserProvider
	UserStorage  domain.UserStorage
}

func NewUserManager(u domain.UserProvider, s domain.UserStorage) (*UserManager, error) {
	return &UserManager{
		u,
		s,
	}, nil
}

func (u UserManager) Login(ctx context.Context, username, password string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	user, err := u.UserProvider.AuthUser(ctx, username, password)
	if err != nil {
		return nil, err
	}
	err = u.UserStorage.SaveUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u UserManager) GetInfo(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	user, err := u.UserStorage.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}
