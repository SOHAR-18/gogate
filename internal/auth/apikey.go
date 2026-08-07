package auth

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type APIKeyValidator struct {
	redis *redis.Client
}

func NewAPIKeyValidator(redisHost, redisPort, redisPassword string) *APIKeyValidator {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})
	return &APIKeyValidator{redis: client}
}

func (v *APIKeyValidator) Validate(apiKey string) (string, error) {
	ctx := context.Background()
	userID, err := v.redis.Get(ctx, "apikey:"+apiKey).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("invalid API key")
	}
	if err != nil {
		return "", fmt.Errorf("redis error: %w", err)
	}
	return userID, nil
}

func (v *APIKeyValidator) Register(apiKey, userID string) error {
	ctx := context.Background()
	return v.redis.Set(ctx, "apikey:"+apiKey, userID, 0).Err()
}
