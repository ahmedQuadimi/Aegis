package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(addr string, password string, db int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

func (c *Client) AllowRequest(ctx context.Context, key string, limit int, frame time.Duration) (bool, error) {
	pipe := c.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, frame)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	if incr.Val() > int64(limit) {
		return false, nil
	}

	return true, nil
}
