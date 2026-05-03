package userProvider

import (
	"YstuPortal/internal/domain"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/charmap"
)

type UserParser struct {
	mu      sync.RWMutex
	clients map[string]*http.Client
	cookies map[string][]*http.Cookie
}

func NewUserParser() *UserParser {
	return &UserParser{
		clients: make(map[string]*http.Client),
		cookies: make(map[string][]*http.Cookie),
	}
}
func (s *UserParser) AuthUser(ctx context.Context, username, password string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// 1. Инициализируем клиент
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Разрешаем редиректы
		},
	}

	// 2. Авторизация
	loginURL := "https://www.ystu.ru/WPROG/auth1.php"
	form := url.Values{}
	form.Set("codeYSTU", "330785001")
	form.Set("login", username)
	form.Set("password", password)

	// Добавляем login1 в кодировке Win1251 вручную
	body := form.Encode() + "&login1=%C2%F5%EE%E4+%BB"

	req, _ := http.NewRequest("POST", loginURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer resp.Body.Close()

	// 3. Переход в личный кабинет для парсинга данных
	// После успешного POST нас редиректит, но мы явно запросим страницу ЛК
	lkURL := "https://www.ystu.ru/WPROG/lk/lkstud2.php"
	lkResp, err := client.Get(lkURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка перехода в ЛК: %w", err)
	}
	defer lkResp.Body.Close()

	// Декодируем кодировку вузовского портала
	utfBody, _ := DecodeWindows1251(lkResp.Body)
	doc, err := goquery.NewDocumentFromReader(utfBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга HTML: %w", err)
	}

	// Проверка: удалось ли войти (ищем ФИО в заголовке h1)
	fullName := strings.TrimSpace(doc.Find("h1").First().Text())
	if fullName == "" {
		return nil, fmt.Errorf("авторизация не удалась: неверный логин или пароль")
	}

	// 4. Заполнение структуры User
	newUser := domain.User{}
	newUser.UserName = username
	newUser.Registered = true

	nameParts := strings.Fields(fullName)
	if len(nameParts) >= 1 {
		newUser.LastName = nameParts[0]
	}
	if len(nameParts) >= 2 {
		newUser.FirstName = nameParts[1]
	}
	if len(nameParts) >= 3 {
		newUser.Patronymic = nameParts[2]
	}

	// Парсим группу
	doc.Find("table tr").Each(func(i int, sel *goquery.Selection) {
		if strings.Contains(sel.Text(), "Группа:") {
			newUser.Group = strings.TrimSpace(sel.Find("td").Last().Text())
		}
	})

	// Сохраняем куки из Jar
	u, _ := url.Parse("https://www.ystu.ru")

	// 5. Сохраняем клиента в кэш хранилища для последующего использования
	s.mu.Lock()
	s.clients[newUser.UserName] = client
	s.cookies[newUser.UserName] = client.Jar.Cookies(u)
	s.mu.Unlock()

	return &newUser, nil
}

func DecodeWindows1251(r io.Reader) (io.Reader, error) {
	return charmap.Windows1251.NewDecoder().Reader(r), nil
}

func (u *UserParser) GetHttpClien(userName string) (*http.Client, error) {
	u.mu.Lock()
	user, ok := u.clients[userName]
	if !ok {
		return nil, fmt.Errorf("нет такого пользователя")
	}
	u.mu.Unlock()
	return user, nil
}

func (s *UserParser) GetEstimations(ctx context.Context) (*map[int]domain.Subject, error) {
	//TODO заполнить получение оценок
	panic("implement me")
}

//TODO: сдеать получение предметов по выбору (вектор, freeminor)
