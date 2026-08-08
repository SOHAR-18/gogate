package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	redis  *redis.Client
	script *redis.Script
}

var luaScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local clear_before = now - window
redis.call("ZREMRANGEBYSCORE", key, 0, clear_before)
local count = redis.call("ZCARD", key)
if count < limit then
  redis.call("ZADD", key, now, now)
  redis.call("EXPIRE", key, window)
  return {1, limit - count - 1, window}
end
local oldest = redis.call("ZRANGE", key, 0, 0, "WITHSCORES")
local retry_after = 0
if #oldest > 0 then
  retry_after = math.ceil(tonumber(oldest[2]) + window - now)
end
return {0, 0, retry_after}
`

type Result struct {
	Allowed    bool
	Remaining  int
	RetryAfter int
}

func NewLimiter(redisHost, redisPort, redisPassword string) *Limiter {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       1,
	})
	return &Limiter{
		redis:  client,
		script: redis.NewScript(luaScript),
	}
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (*Result, error) {
	now := time.Now().UnixMilli()
	windowMs := int64(window / time.Millisecond)

	vals, err := l.script.Run(ctx, l.redis,
		[]string{key},
		limit,
		windowMs,
		now,
	).Int64Slice()

	if err != nil {
		return &Result{Allowed: true}, nil
	}

	return &Result{
		Allowed:    vals[0] == 1,
		Remaining:  int(vals[1]),
		RetryAfter: int(vals[2]),
	}, nil
}
