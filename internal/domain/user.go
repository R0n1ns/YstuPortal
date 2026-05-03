package domain

import (
	"context"

	"github.com/google/uuid"
)

//type Group struct {
//	Id    uuid.UUID
//	Title string
//}

type User struct {
	Id         uuid.UUID `json:"id"`
	FirstName  string    `json:"firstname"`
	LastName   string    `json:"lastname"`
	Patronymic string    `json:"patronymic"`
	UserName   string    `json:"username"`
	Mail       string    `json:"mail"`
	Password   string    `json:"password"`
	Registered bool      `json:"registered"`
	//Group       Group           `json:"group"`
	Group       string          `json:"group"`
	Estimations map[int]Subject `json:"estimations"`
}

type UserPort interface {
	GetFIO() (FirstName, LastName, Patronymic string)
	GetId() uuid.UUID
	IsReg() bool
	//GetGroup() Group
	GetGroup() string
	GetEstimations() map[int]Subject
	SetGroup(string) bool
}

type UserProvider interface {
	AuthUser(ctx context.Context, username, password string) (*User, error)
	GetEstimations(ctx context.Context) (*map[int]Subject, error)
}

//TODO: реализовать добавление в группу :при регистрации пользователя в приложении , проверять его группу и составлять автомтически
