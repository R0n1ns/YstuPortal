package userProvider

import (
	"YstuPortal/internal/domain"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
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
func (u *UserParser) Close() {
	for _, client := range u.clients {
		client.CloseIdleConnections()
	}
}

func (s *UserParser) GetUser(ctx context.Context, username, password string) (*domain.User, error) {
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
	defer func() { _ = resp.Body.Close() }()

	// 3. Переход в личный кабинет для парсинга данных
	// После успешного POST нас редиректит, но мы явно запросим страницу ЛК
	lkURL := "https://www.ystu.ru/WPROG/lk/lkstud2.php"
	lkResp, err := client.Get(lkURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка перехода в ЛК: %w", err)
	}
	defer func() { _ = lkResp.Body.Close() }()

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

func (u *UserParser) GetHttpClient(userName string) (*http.Client, error) {
	u.mu.Lock()
	user, ok := u.clients[userName]
	if !ok {
		return nil, fmt.Errorf("нет такого пользователя")
	}
	u.mu.Unlock()
	return user, nil
}

func (s *UserParser) GetEstimations(ctx context.Context) (*[]domain.Subject, error) {
	//fmt.Println(1)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	userName, ok := ctx.Value("UserName").(string)
	if !ok || strings.TrimSpace(userName) == "" {
		return nil, fmt.Errorf("не задан UserName в контексте")
	}

	s.mu.RLock()
	client, ok := s.clients[userName]
	s.mu.RUnlock()
	if !ok || client == nil {
		return nil, fmt.Errorf("нет клиента для пользователя")
	}

	estimationsURL := "https://www.ystu.ru/WPROG/lk/lkstud_oc.php"
	resp, err := client.Get(estimationsURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса оценок: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	utfBody, _ := DecodeWindows1251(resp.Body)
	doc, err := goquery.NewDocumentFromReader(utfBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга HTML: %w", err)
	}

	normalize := func(text string) string {
		text = strings.ReplaceAll(text, "\u00a0", " ")
		text = strings.ToLower(strings.TrimSpace(text))
		text = strings.ReplaceAll(text, "\n", " ")
		text = strings.ReplaceAll(text, "\r", " ")
		text = strings.ReplaceAll(text, "\t", " ")
		text = strings.Join(strings.Fields(text), "")
		return text
	}

	extractInt := func(text string) int {
		var b strings.Builder
		for _, r := range text {
			if r >= '0' && r <= '9' {
				_ = b.WriteByte(byte(r))
			}
		}
		if b.Len() == 0 {
			return 0
		}
		value, err := strconv.Atoi(b.String())
		if err != nil {
			return 0
		}
		return value
	}

	headerIndex := map[string]int{}
	var tableWithGrades *goquery.Selection
	var headerRow *goquery.Selection

	doc.Find("table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		var found bool
		table.Find("tr").EachWithBreak(func(_ int, row *goquery.Selection) bool {
			cells := row.Find("th, td")
			if cells.Length() == 0 {
				return true
			}
			localIndex := map[string]int{}
			cells.Each(func(i int, cell *goquery.Selection) {
				cellText := normalize(cell.Text())
				switch {
				case strings.Contains(cellText, "№"):
					localIndex["num"] = i
				case strings.Contains(cellText, "курс"):
					localIndex["course"] = i
				case strings.Contains(cellText, "семестр"):
					localIndex["semester"] = i
				case strings.Contains(cellText, "наименовани"):
					localIndex["title"] = i
				case strings.Contains(cellText, "видконтрол"):
					localIndex["type"] = i
				case strings.Contains(cellText, "зед"):
					localIndex["zed"] = i
				case strings.Contains(cellText, "балл"):
					localIndex["score"] = i
				case strings.Contains(cellText, "оценка"):
					localIndex["evaluation"] = i
				}
			})
			if _, ok := localIndex["title"]; ok {
				if _, ok := localIndex["evaluation"]; ok {
					found = true
					headerIndex = localIndex
					headerRow = row
					return false
				}
			}
			return true
		})
		if found {
			tableWithGrades = table
			return false
		}
		return true
	})

	if tableWithGrades == nil {
		return nil, fmt.Errorf("таблица оценок не найдена")
	}

	getCellText := func(cells *goquery.Selection, key string) string {
		idx, ok := headerIndex[key]
		if !ok || idx >= cells.Length() {
			return ""
		}
		return strings.TrimSpace(cells.Eq(idx).Text())
	}

	result := []domain.Subject{}
	rowCounter := 0
	tableWithGrades.Find("tr").Each(func(_ int, row *goquery.Selection) {
		if headerRow != nil && row.IsSelection(headerRow) {
			return
		}
		cells := row.Find("td")
		if cells.Length() == 0 {
			return
		}

		numValue := extractInt(getCellText(cells, "num"))
		if numValue == 0 {
			rowCounter++
			numValue = rowCounter
		}

		title := strings.TrimSpace(getCellText(cells, "title"))
		diploma := strings.Contains(title, "*")
		title = strings.TrimSpace(strings.ReplaceAll(title, "*", ""))

		subject := domain.Subject{
			Course:        extractInt(getCellText(cells, "course")),
			Semester:      extractInt(getCellText(cells, "semester")),
			Title:         title,
			TypeOfControl: strings.TrimSpace(getCellText(cells, "type")),
			Zed:           extractInt(getCellText(cells, "zed")),
			Mark:          strings.TrimSpace(getCellText(cells, "score")),
			Evaluation:    strings.TrimSpace(getCellText(cells, "evaluation")),
			Diploma:       diploma,
		}

		result = append(result, subject)
	})
	//fmt.Println(result)

	return &result, nil
}
