package userProvider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/R0n1ns/YstuPortal/internal/domain"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/charmap"
)

const (
	loginPath   = "/WPROG/auth1.php"
	profilePath = "/WPROG/lk/lkstud2.php"
	gradesPath  = "/WPROG/lk/lkstud_oc.php"
)

type userSession struct {
	client   *http.Client
	lastUsed time.Time
}

type UserParser struct {
	mu         sync.RWMutex
	sessions   map[string]*userSession
	baseURL    string
	portalCode string
	timeout    time.Duration
	sessionTTL time.Duration
}

func NewUserParser(baseURL, portalCode string, timeout, sessionTTL time.Duration) *UserParser {
	return &UserParser{
		sessions:   make(map[string]*userSession),
		baseURL:    strings.TrimRight(baseURL, "/"),
		portalCode: portalCode,
		timeout:    timeout,
		sessionTTL: sessionTTL,
	}
}

func (p *UserParser) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for userName, session := range p.sessions {
		session.client.CloseIdleConnections()
		delete(p.sessions, userName)
	}
}

func (p *UserParser) CloseSession(userName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if session, ok := p.sessions[userName]; ok {
		session.client.CloseIdleConnections()
		delete(p.sessions, userName)
	}
}

func (p *UserParser) GetUser(ctx context.Context, username, password string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	client := &http.Client{Jar: jar, Timeout: p.timeout}

	form := url.Values{}
	form.Set("codeYSTU", p.portalCode)
	form.Set("login", username)
	form.Set("password", password)
	body := form.Encode() + "&login1=%C2%F5%EE%E4+%BB"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+loginPath, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request: %w", err)
	}
	if err := closeResponse(resp, "login"); err != nil {
		return nil, err
	}

	profileReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+profilePath, nil)
	if err != nil {
		return nil, fmt.Errorf("create profile request: %w", err)
	}
	profileResp, err := client.Do(profileReq)
	if err != nil {
		return nil, fmt.Errorf("profile request: %w", err)
	}
	defer func() { _ = profileResp.Body.Close() }()
	if err := checkStatus(profileResp, "profile"); err != nil {
		return nil, err
	}

	doc, err := decodeDocument(profileResp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	fullName := strings.TrimSpace(doc.Find("h1").First().Text())
	if fullName == "" {
		return nil, fmt.Errorf("%w: profile page does not contain a user name", domain.ErrInvalidCredentials)
	}

	user := &domain.User{UserName: username, Registered: true, Role: "student"}
	nameParts := strings.Fields(fullName)
	if len(nameParts) > 0 {
		user.LastName = nameParts[0]
	}
	if len(nameParts) > 1 {
		user.FirstName = nameParts[1]
	}
	if len(nameParts) > 2 {
		user.Patronymic = nameParts[2]
	}
	doc.Find("table tr").Each(func(_ int, row *goquery.Selection) {
		if strings.Contains(row.Text(), "Группа:") {
			user.Group = strings.TrimSpace(row.Find("td").Last().Text())
		}
	})

	p.mu.Lock()
	p.pruneExpiredLocked(time.Now())
	if previous, ok := p.sessions[username]; ok {
		previous.client.CloseIdleConnections()
	}
	p.sessions[username] = &userSession{client: client, lastUsed: time.Now()}
	p.mu.Unlock()

	return user, nil
}

func (p *UserParser) GetGrades(ctx context.Context, userName string) ([]domain.Subject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.pruneExpiredLocked(time.Now())
	session, ok := p.sessions[userName]
	if ok {
		session.lastUsed = time.Now()
	}
	p.mu.Unlock()
	if !ok {
		return nil, domain.ErrSessionNotFound
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+gradesPath, nil)
	if err != nil {
		return nil, fmt.Errorf("create grades request: %w", err)
	}
	resp, err := session.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("grades request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := checkStatus(resp, "grades"); err != nil {
		return nil, err
	}

	doc, err := decodeDocument(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse grades: %w", err)
	}
	return parseGrades(doc)
}

func (p *UserParser) pruneExpiredLocked(now time.Time) {
	if p.sessionTTL <= 0 {
		return
	}
	for userName, session := range p.sessions {
		if now.Sub(session.lastUsed) > p.sessionTTL {
			session.client.CloseIdleConnections()
			delete(p.sessions, userName)
		}
	}
}

func closeResponse(resp *http.Response, operation string) error {
	defer func() { _ = resp.Body.Close() }()
	if err := checkStatus(resp, operation); err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func checkStatus(resp *http.Response, operation string) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%s: upstream returned HTTP %d", operation, resp.StatusCode)
	}
	return nil
}

func decodeDocument(body io.Reader) (*goquery.Document, error) {
	decoded := charmap.Windows1251.NewDecoder().Reader(body)
	return goquery.NewDocumentFromReader(decoded)
}

func parseGrades(doc *goquery.Document) ([]domain.Subject, error) {
	headerIndex := map[string]int{}
	var gradesTable *goquery.Selection
	var headerRow *goquery.Selection

	doc.Find("table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		found := false
		table.Find("tr").EachWithBreak(func(_ int, row *goquery.Selection) bool {
			localIndex := gradeHeaderIndexes(row.Find("th, td"))
			if _, titleOK := localIndex["title"]; titleOK {
				if _, evaluationOK := localIndex["evaluation"]; evaluationOK {
					headerIndex, headerRow, found = localIndex, row, true
					return false
				}
			}
			return true
		})
		if found {
			gradesTable = table
			return false
		}
		return true
	})
	if gradesTable == nil {
		return nil, fmt.Errorf("grades table not found")
	}

	cellText := func(cells *goquery.Selection, key string) string {
		index, ok := headerIndex[key]
		if !ok || index >= cells.Length() {
			return ""
		}
		return strings.TrimSpace(cells.Eq(index).Text())
	}

	grades := make([]domain.Subject, 0)
	gradesTable.Find("tr").Each(func(_ int, row *goquery.Selection) {
		if headerRow != nil && row.IsSelection(headerRow) {
			return
		}
		cells := row.Find("td")
		if cells.Length() == 0 {
			return
		}
		title := cellText(cells, "title")
		diploma := strings.Contains(title, "*")
		title = strings.TrimSpace(strings.ReplaceAll(title, "*", ""))
		if title == "" {
			return
		}
		grades = append(grades, domain.Subject{
			Course:        extractInt(cellText(cells, "course")),
			Semester:      extractInt(cellText(cells, "semester")),
			Title:         title,
			TypeOfControl: cellText(cells, "type"),
			Zed:           extractInt(cellText(cells, "zed")),
			Mark:          cellText(cells, "score"),
			Evaluation:    cellText(cells, "evaluation"),
			Diploma:       diploma,
		})
	})
	return grades, nil
}

func gradeHeaderIndexes(cells *goquery.Selection) map[string]int {
	indexes := map[string]int{}
	cells.Each(func(index int, cell *goquery.Selection) {
		text := normalize(cell.Text())
		switch {
		case strings.Contains(text, "курс"):
			indexes["course"] = index
		case strings.Contains(text, "семестр"):
			indexes["semester"] = index
		case strings.Contains(text, "наименовани"):
			indexes["title"] = index
		case strings.Contains(text, "видконтрол"):
			indexes["type"] = index
		case strings.Contains(text, "зед"):
			indexes["zed"] = index
		case strings.Contains(text, "балл"):
			indexes["score"] = index
		case strings.Contains(text, "оценка"):
			indexes["evaluation"] = index
		}
	})
	return indexes
}

func normalize(value string) string {
	value = strings.NewReplacer("\u00a0", " ", "\n", " ", "\r", " ", "\t", " ").Replace(value)
	return strings.ToLower(strings.Join(strings.Fields(value), ""))
}

func extractInt(value string) int {
	var digits strings.Builder
	for _, symbol := range value {
		if symbol >= '0' && symbol <= '9' {
			digits.WriteRune(symbol)
		}
	}
	result, _ := strconv.Atoi(digits.String())
	return result
}
