package logic

import (
	"YstuPortal/internal/domain"
	"context"
	"errors"
	"testing"
	"time"
)

type stubCache struct {
	data []domain.Subject
	ok   bool
}

func (c *stubCache) Get(ctx context.Context, userName string) ([]domain.Subject, bool, error) {
	return c.data, c.ok, nil
}

func (c *stubCache) Set(ctx context.Context, userName string, data []domain.Subject, ttl time.Duration) error {
	return nil
}

type stubStorage struct {
	getCalled bool
}

func (s *stubStorage) SaveUser(ctx context.Context, user *domain.User) error {
	return nil
}

func (s *stubStorage) GetUser(ctx context.Context, userName string) (*domain.User, error) {
	s.getCalled = true
	return nil, errors.New("storage should not be called")
}

type stubProvider struct{}

func (p *stubProvider) GetUser(ctx context.Context, username, password string) (*domain.User, error) {
	return nil, errors.New("provider should not be called")
}

func (p *stubProvider) GetEstimations(ctx context.Context) (*[]domain.Subject, error) {
	return nil, errors.New("provider should not be called")
}

func TestGetEstimationsUsesCache(t *testing.T) {
	cached := []domain.Subject{{Title: "Math"}}
	cache := &stubCache{data: cached, ok: true}
	storage := &stubStorage{}
	provider := &stubProvider{}

	manager, err := NewUserManagerWithCache(provider, storage, cache, time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, err := manager.GetEstimations(context.Background(), "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(data) != 1 || data[0].Title != "Math" {
		t.Fatalf("unexpected cache data")
	}
	if storage.getCalled {
		t.Fatalf("storage should not be called when cache hit")
	}
}
