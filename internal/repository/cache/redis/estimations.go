package redis

import (
	"YstuPortal/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const estimationsKeyPrefix = "estimations:"

type EstimationsCache struct {
	client *redis.Client
}

func NewEstimationsCache(redisURL string) (*EstimationsCache, error) {
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

	return &EstimationsCache{client: client}, nil
}

func (c *EstimationsCache) Get(ctx context.Context, userName string) ([]domain.Subject, bool, error) {
	value, err := c.client.Get(ctx, estimationsKeyPrefix+userName).Result()
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

func (c *EstimationsCache) Set(ctx context.Context, userName string, data []domain.Subject, ttl time.Duration) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, estimationsKeyPrefix+userName, payload, ttl).Err()
}

func (c *EstimationsCache) Close() error {
	return c.client.Close()
}
