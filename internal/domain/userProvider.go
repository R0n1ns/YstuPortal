package domain

import (
	"context"
)

type User struct {
	FirstName  string `json:"firstname"`
	LastName   string `json:"lastname"`
	Patronymic string `json:"patronymic"`
	UserName   string `json:"username"`
	Mail       string `json:"mail"`
	Password   string `json:"password"`
	Registered bool   `json:"registered"`
	//Group       Group           `json:"group"`
	Group       string    `json:"group"`
	Estimations []Subject `json:"estimations"`
}

type UserPort interface {
	GetFIO() (FirstName, LastName, Patronymic string)
	GetUserName() string
	IsReg() bool
	//GetGroup() Group
	GetGroup() string
	GetEstimations() []Subject
	SetGroup(string) bool
}

type UserProvider interface {
	GetUser(ctx context.Context, username, password string) (*User, error)
	GetEstimations(ctx context.Context) (*[]Subject, error)
}

//TODO: реализовать добавление в группу :при регистрации пользователя в приложении , проверять его группу и составлять автомтически
