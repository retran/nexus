// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitConfig defines rate limiting configuration for an endpoint.
type RateLimitConfig struct {
	KeyFunc  func(*http.Request) string
	Requests int
	Window   time.Duration
}

// RateLimiter handles rate limiting using Redis.
type RateLimiter struct {
	redis  *redis.Client
	config RateLimitConfig
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(redisClient *redis.Client, config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		config: config,
	}
}

// DefaultKeyFunc generates a rate limit key based on IP address.
func DefaultKeyFunc(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// UserKeyFunc generates a rate limit key based on authenticated user.
func UserKeyFunc(r *http.Request) string {
	if authInfo, ok := r.Context().Value(AuthContextKey).(*AuthInfo); ok {
		return fmt.Sprintf("user:%s", authInfo.UserID.String())
	}
	return DefaultKeyFunc(r)
}

// Allow checks if the request is allowed under the rate limit.
func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, int, time.Time, error) {
	if rl.redis == nil {
		log.Println("Warning: Redis not available, rate limiting disabled")
		return true, rl.config.Requests, time.Now().Add(rl.config.Window), nil
	}

	now := time.Now()
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, now.Unix()/int64(rl.config.Window.Seconds()))

	pipe := rl.redis.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, rl.config.Window)

	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("Warning: Redis error during rate limiting: %v", err)
		return true, rl.config.Requests, now.Add(rl.config.Window), nil
	}

	count := int(incr.Val())
	remaining := rl.config.Requests - count
	if remaining < 0 {
		remaining = 0
	}

	resetTime := now.Truncate(rl.config.Window).Add(rl.config.Window)
	allowed := count <= rl.config.Requests

	return allowed, remaining, resetTime, nil
}

// Middleware returns an HTTP middleware that applies rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.config.KeyFunc(r)
		allowed, remaining, resetTime, err := rl.Allow(r.Context(), key)

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.config.Requests))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if err != nil {
			log.Printf("Rate limit error: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			w.Header().Set("Retry-After", strconv.FormatInt(int64(time.Until(resetTime).Seconds()), 10))
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
