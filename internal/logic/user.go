package logic

import (
	"YstuPortal/internal/domain"
	"context"
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

func (u UserManager) GetInfo(ctx context.Context, userName string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	user, err := u.UserStorage.GetUser(ctx, userName)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u UserManager) GetEstimations(ctx context.Context, userName string) (map[int]domain.Subject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	user, err := u.UserStorage.GetUser(ctx, userName)
	if err != nil {
		return nil, err
	}
	return user.Estimations, nil
}
