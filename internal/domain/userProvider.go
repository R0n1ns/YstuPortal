package domain

import (
	"context"
)

type User struct {
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Patronymic string    `json:"patronymic,omitempty"`
	UserName   string    `json:"username"`
	Mail       string    `json:"mail,omitempty"`
	Registered bool      `json:"registered"`
	Role       string    `json:"role"`
	Group      string    `json:"group,omitempty"`
	Grades     []Subject `json:"grades,omitempty"`
}

type UserProvider interface {
	GetUser(ctx context.Context, username, password string) (*User, error)
	GetGrades(ctx context.Context, username string) ([]Subject, error)
}
