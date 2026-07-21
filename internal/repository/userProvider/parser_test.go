package userProvider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/R0n1ns/YstuPortal/internal/domain"

	"golang.org/x/text/encoding/charmap"
)

func TestUserParserLoginAndGrades(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case loginPath:
			if err := request.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if request.FormValue("login") != "student" || request.FormValue("password") != "secret1" {
				t.Fatalf("unexpected credentials")
			}
			http.SetCookie(response, &http.Cookie{Name: "session", Value: "active"})
			response.WriteHeader(http.StatusOK)
		case profilePath:
			requireSession(t, request)
			writeWindows1251(t, response, `<html><h1>Иванов Иван Иванович</h1><table><tr><td>Группа:</td><td>ИВТ-01</td></tr></table></html>`)
		case gradesPath:
			requireSession(t, request)
			writeWindows1251(t, response, `<table><tr><th>Курс</th><th>Семестр</th><th>Наименование</th><th>Вид контроля</th><th>ЗЕТ</th><th>Балл</th><th>Оценка</th></tr><tr><td>2</td><td>3</td><td>Алгоритмы *</td><td>Экзамен</td><td>4</td><td>91</td><td>Отлично</td></tr></table>`)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	parser := NewUserParser(server.URL, "test-code", time.Second, time.Hour)
	defer parser.Close()
	user, err := parser.GetUser(context.Background(), "student", "secret1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if user.FirstName != "Иван" || user.LastName != "Иванов" || user.Group != "ИВТ-01" {
		t.Fatalf("unexpected user: %+v", user)
	}

	grades, err := parser.GetGrades(context.Background(), user.UserName)
	if err != nil {
		t.Fatalf("grades: %v", err)
	}
	if len(grades) != 1 || grades[0].Title != "Алгоритмы" || !grades[0].Diploma || grades[0].Mark != "91" {
		t.Fatalf("unexpected grades: %+v", grades)
	}

	parser.CloseSession(user.UserName)
	_, err = parser.GetGrades(context.Background(), user.UserName)
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func requireSession(t *testing.T, request *http.Request) {
	t.Helper()
	if _, err := request.Cookie("session"); err != nil {
		t.Fatalf("session cookie is missing: %v", err)
	}
}

func writeWindows1251(t *testing.T, response http.ResponseWriter, value string) {
	t.Helper()
	encoded, err := charmap.Windows1251.NewEncoder().Bytes([]byte(value))
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}
	response.WriteHeader(http.StatusOK)
	if _, err := response.Write(encoded); err != nil {
		t.Fatalf("write response: %v", err)
	}
}
