package domain

import (
	"context"
)

type UserStorage interface {
	SaveUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, userName string) (*User, error)
}
