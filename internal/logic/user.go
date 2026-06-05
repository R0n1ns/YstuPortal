package logic

import (
	"YstuPortal/internal/domain"
	"context"
	"time"
)

type UserManagerType interface {
	Login(ctx context.Context, username, password string) (*domain.User, error)
	GetInfo(ctx context.Context, userName string) (*domain.User, error)
	GetEstimations(ctx context.Context, userName string) ([]domain.Subject, error)
}

type EstimationsCache interface {
	Get(ctx context.Context, userName string) ([]domain.Subject, bool, error)
	Set(ctx context.Context, userName string, data []domain.Subject, ttl time.Duration) error
}

type UserManager struct {
	UserProvider domain.UserProvider
	UserStorage  domain.UserStorage
	cache        EstimationsCache
	cacheTTL     time.Duration
}

func NewUserManager(u domain.UserProvider, s domain.UserStorage) (*UserManager, error) {
	return &UserManager{
		UserProvider: u,
		UserStorage:  s,
	}, nil
}

func NewUserManagerWithCache(u domain.UserProvider, s domain.UserStorage, cache EstimationsCache, cacheTTL time.Duration) (*UserManager, error) {
	return &UserManager{
		UserProvider: u,
		UserStorage:  s,
		cache:        cache,
		cacheTTL:     cacheTTL,
	}, nil
}

func (u UserManager) Login(ctx context.Context, username, password string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	user, err := u.UserProvider.GetUser(ctx, username, password)
	if err != nil {
		return nil, err
	}
	if user.Role == "" {
		user.Role = "student"
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
	if len(user.Estimations) == 0 {
		estimations, err := u.UserProvider.GetEstimations(ctx)
		if err != nil {
			return nil, err
		}
		user.Estimations = *estimations
		err = u.UserStorage.SaveUser(ctx, user)
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (u UserManager) GetEstimations(ctx context.Context, userName string) ([]domain.Subject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if u.cache != nil {
		if cached, ok, err := u.cache.Get(ctx, userName); err == nil && ok {
			return cached, nil
		}
	}
	user, err := u.UserStorage.GetUser(ctx, userName)
	if err != nil {
		return nil, err
	}
	if len(user.Estimations) == 0 {
		estimations, err := u.UserProvider.GetEstimations(ctx)
		if err != nil {
			return nil, err
		}
		user.Estimations = *estimations
		err = u.UserStorage.SaveUser(ctx, user)
		if err != nil {
			return nil, err
		}
	}
	if u.cache != nil && len(user.Estimations) > 0 {
		_ = u.cache.Set(ctx, userName, user.Estimations, u.cacheTTL)
	}
	return user.Estimations, nil
}
