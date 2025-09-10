package rate

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Limiter handles rate limiting using Redis
type Limiter struct {
	client   *redis.Client
	capacity int
	refill   time.Duration
}

// NewLimiter creates a new rate limiter
func NewLimiter(client *redis.Client, capacity int, refill time.Duration) *Limiter {
	return &Limiter{
		client:   client,
		capacity: capacity,
		refill:   refill,
	}
}

// Allow checks if a request is allowed for a user
func (l *Limiter) Allow(ctx context.Context, userID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("rate:%s", userID.String())

	// Lua script for atomic rate limiting
	script := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_per_sec = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1]) or capacity
		local last_refill = tonumber(bucket[2]) or now
		
		-- Calculate time elapsed since last refill
		local elapsed = (now - last_refill) / 1000.0
		local new_tokens = math.min(capacity, tokens + elapsed * refill_per_sec)
		
		-- Check if we can allow the request
		if new_tokens < 1 then
			-- Update the bucket with current time
			redis.call('HMSET', key, 'tokens', new_tokens, 'last_refill', now)
			redis.call('EXPIRE', key, 60)
			return 0
		end
		
		-- Allow the request and decrement tokens
		new_tokens = new_tokens - 1
		redis.call('HMSET', key, 'tokens', new_tokens, 'last_refill', now)
		redis.call('EXPIRE', key, 60)
		return 1
	`

	refillPerSec := float64(l.capacity) / l.refill.Seconds()
	now := time.Now().UnixMilli()

	result, err := l.client.Eval(ctx, script, []string{key}, l.capacity, refillPerSec, now).Result()
	if err != nil {
		return false, fmt.Errorf("failed to execute rate limit script: %w", err)
	}

	allowed := result.(int64) == 1
	return allowed, nil
}

// GetRemainingTokens returns the number of remaining tokens for a user
func (l *Limiter) GetRemainingTokens(ctx context.Context, userID uuid.UUID) (int, error) {
	key := fmt.Sprintf("rate:%s", userID.String())

	// Lua script to get remaining tokens
	script := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_per_sec = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1]) or capacity
		local last_refill = tonumber(bucket[2]) or now
		
		-- Calculate time elapsed since last refill
		local elapsed = (now - last_refill) / 1000.0
		local new_tokens = math.min(capacity, tokens + elapsed * refill_per_sec)
		
		return math.floor(new_tokens)
	`

	refillPerSec := float64(l.capacity) / l.refill.Seconds()
	now := time.Now().UnixMilli()

	result, err := l.client.Eval(ctx, script, []string{key}, l.capacity, refillPerSec, now).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get remaining tokens: %w", err)
	}

	return int(result.(int64)), nil
}

// Reset resets the rate limit for a user
func (l *Limiter) Reset(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("rate:%s", userID.String())
	return l.client.Del(ctx, key).Err()
}
