package db

import (
	"YstuPortal/internal/domain"
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type UserStorage struct {
	mu    sync.RWMutex
	users map[uuid.UUID]*domain.User
}

func NewUserStorage() *UserStorage {
	return &UserStorage{
		users: make(map[uuid.UUID]*domain.User),
	}
}

func (u *UserStorage) GetUser(ctx context.Context, uuid uuid.UUID) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	u.mu.Lock()
	user, ok := u.users[uuid]
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
	u.users[user.Id] = user
	u.mu.Unlock()
	return nil
}

//TODO: сделать сохранение пользователя в бд и для быстрокого получения хеширования в redis
