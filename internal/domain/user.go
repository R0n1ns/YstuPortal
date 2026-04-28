package domain

import (
	"context"

	"github.com/google/uuid"
)

type User struct {
	Id         uuid.UUID `json:"id"`
	FirstName  string    `json:"firstname"`
	LastName   string    `json:"lastname"`
	Patronymic string    `json:"patronymic"`
	UserName   string    `json:"username"`
	Mail       string    `json:"mail"`
	Password   string    `json:"password"`
	Registered bool      `json:"registered"`
	Group      string    `json:"group"`
}

type UserPort interface {
	GetFIO() (FirstName, LastName, Patronymic string)
	GetId() uuid.UUID
	IsReg() bool
	GetGroup() string
}

type UserProvider interface {
	AuthUser(ctx context.Context, username, password string) (*UserPort, error)
}
