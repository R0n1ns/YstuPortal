package logic

import (
	"context"
	"errors"
	"time"

	"github.com/R0n1ns/YstuPortal/internal/domain"
)

type UserManagerType interface {
	Login(ctx context.Context, username, password string) (*domain.User, error)
	GetInfo(ctx context.Context, userName string) (*domain.User, error)
	GetGrades(ctx context.Context, userName string) ([]domain.Subject, error)
	Logout(userName string)
}

type GradesCache interface {
	Get(ctx context.Context, userName string) ([]domain.Subject, bool, error)
	Set(ctx context.Context, userName string, data []domain.Subject, ttl time.Duration) error
}

type UserManager struct {
	UserProvider domain.UserProvider
	UserStorage  domain.UserStorage
	cache        GradesCache
	cacheTTL     time.Duration
}

func NewUserManager(u domain.UserProvider, s domain.UserStorage) (*UserManager, error) {
	if u == nil || s == nil {
		return nil, errors.New("user provider and storage are required")
	}
	return &UserManager{
		UserProvider: u,
		UserStorage:  s,
	}, nil
}

func NewUserManagerWithCache(u domain.UserProvider, s domain.UserStorage, cache GradesCache, cacheTTL time.Duration) (*UserManager, error) {
	if u == nil || s == nil {
		return nil, errors.New("user provider and storage are required")
	}
	if cache == nil {
		return nil, errors.New("cache is required")
	}
	if cacheTTL <= 0 {
		return nil, errors.New("cache ttl must be positive")
	}
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
	return u.UserStorage.GetUser(ctx, userName)
}

func (u UserManager) GetGrades(ctx context.Context, userName string) ([]domain.Subject, error) {
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
	if len(user.Grades) == 0 {
		grades, err := u.UserProvider.GetGrades(ctx, userName)
		if err != nil {
			return nil, err
		}
		user.Grades = grades
		err = u.UserStorage.SaveUser(ctx, user)
		if err != nil {
			return nil, err
		}
	}
	if u.cache != nil && len(user.Grades) > 0 {
		_ = u.cache.Set(ctx, userName, user.Grades, u.cacheTTL)
	}
	return user.Grades, nil
}

type sessionCloser interface {
	CloseSession(userName string)
}

func (u UserManager) Logout(userName string) {
	if closer, ok := u.UserProvider.(sessionCloser); ok {
		closer.CloseSession(userName)
	}
}
