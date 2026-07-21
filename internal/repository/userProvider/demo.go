package userProvider

import (
	"context"
	"fmt"

	"github.com/R0n1ns/YstuPortal/internal/domain"
)

type DemoProvider struct{}

func NewDemoProvider() *DemoProvider {
	return &DemoProvider{}
}

func (p *DemoProvider) GetUser(ctx context.Context, username, password string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	role := "student"
	switch {
	case username == "demo" && password == "demo123":
	case username == "demo-admin" && password == "admin123":
		role = "admin"
	default:
		return nil, fmt.Errorf("%w: demo account does not match", domain.ErrInvalidCredentials)
	}
	return &domain.User{
		FirstName:  "Иван",
		LastName:   "Иванов",
		Patronymic: "Иванович",
		UserName:   username,
		Mail:       username + "@example.test",
		Registered: true,
		Role:       role,
		Group:      "ИВТ-01",
	}, nil
}

func (p *DemoProvider) GetGrades(ctx context.Context, _ string) ([]domain.Subject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []domain.Subject{
		{Course: 2, Semester: 3, Title: "Алгоритмы и структуры данных", TypeOfControl: "Экзамен", Zed: 4, Mark: "91", Evaluation: "Отлично"},
		{Course: 2, Semester: 3, Title: "Базы данных", TypeOfControl: "Зачёт", Zed: 3, Mark: "зачтено", Evaluation: "Зачтено"},
		{Course: 2, Semester: 4, Title: "Разработка web-приложений", TypeOfControl: "Курсовая работа", Zed: 5, Mark: "88", Evaluation: "Хорошо", Diploma: true},
	}, nil
}

func (p *DemoProvider) CloseSession(string) {}
