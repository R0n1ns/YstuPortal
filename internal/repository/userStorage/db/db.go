package db

import (
	"YstuPortal/internal/domain"
	"context"
	"fmt"
	"sync"
)

type UserStorage struct {
	mu    sync.RWMutex
	users map[string]*domain.User
}

func NewUserStorage() *UserStorage {
	return &UserStorage{
		users: make(map[string]*domain.User),
	}
}

func (u *UserStorage) GetUser(ctx context.Context, userName string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	u.mu.Lock()
	user, ok := u.users[userName]
	if !ok {
		return nil, fmt.Errorf("нет такого пользователя")
	}
	u.mu.Unlock()
	return user, nil
}
func (u *UserStorage) SaveUser(ctx context.Context, user *domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	u.mu.Lock()
	u.users[user.UserName] = user
	u.mu.Unlock()
	return nil
}

//TODO: сделать сохранение пользователя в бд и для быстрокого получения хеширования в redis
