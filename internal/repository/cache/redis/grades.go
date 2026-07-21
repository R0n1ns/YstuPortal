package redis

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/R0n1ns/YstuPortal/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

const gradesKeyPrefix = "grades:"

type GradesCache struct {
	client *redis.Client
}

func NewGradesCache(redisURL string) (*GradesCache, error) {
	if redisURL == "" {
		return nil, errors.New("redis url is empty")
	}

	options, err := redis.ParseURL(redisURL)
	if err != nil {
		options = &redis.Options{Addr: redisURL}
	}

	client := redis.NewClient(options)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &GradesCache{client: client}, nil
}

func (c *GradesCache) Get(ctx context.Context, userName string) ([]domain.Subject, bool, error) {
	value, err := c.client.Get(ctx, gradesKeyPrefix+userName).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var data []domain.Subject
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return nil, false, err
	}

	return data, true, nil
}

func (c *GradesCache) Set(ctx context.Context, userName string, data []domain.Subject, ttl time.Duration) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, gradesKeyPrefix+userName, payload, ttl).Err()
}

func (c *GradesCache) Close() error {
	return c.client.Close()
}

func (c *GradesCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
