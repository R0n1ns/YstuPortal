package userProvider

import (
	"context"
	"errors"
	"testing"

	"github.com/R0n1ns/YstuPortal/internal/domain"
)

func TestDemoProvider(t *testing.T) {
	provider := NewDemoProvider()
	user, err := provider.GetUser(context.Background(), "demo", "demo123")
	if err != nil {
		t.Fatalf("login demo user: %v", err)
	}
	if user.Role != "student" || user.UserName != "demo" {
		t.Fatalf("unexpected user: %+v", user)
	}
	grades, err := provider.GetGrades(context.Background(), user.UserName)
	if err != nil || len(grades) < 3 {
		t.Fatalf("unexpected grades: %v, %+v", err, grades)
	}
	_, err = provider.GetUser(context.Background(), "demo", "wrong-password")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}
