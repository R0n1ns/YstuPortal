package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserStorage interface {
	SaveUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
}
