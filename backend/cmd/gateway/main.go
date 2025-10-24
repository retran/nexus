// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/retran/nexus/backend/internal/api/rest"
)

func main() {
	cfg := rest.Config{
		Port:               8080,
		Host:               "0.0.0.0",
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       10 * time.Second,
		ShutdownTimeout:    30 * time.Second,
		GraphQLEndpoint:    getEnv("GRAPHQL_ENDPOINT", "http://localhost:8081/graphql"),
		AllowedOrigins:     getAllowedOrigins(),
		DatabaseURL:        getDatabaseURL(),
		RedisHost:          getEnv("REDIS_HOST", "localhost"),
		RedisPort:          getEnvInt("REDIS_PORT", 6379),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("REDIS_DB", 0),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://api.nexus.local/api/auth/google/callback"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production-use-a-strong-secret"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://nexus.local"),
		TemporalHost:       getEnv("TEMPORAL_HOST", "localhost:7233"),
		TemporalNamespace:  getEnv("TEMPORAL_NAMESPACE", "default"),
		TemporalTaskQueue:  getEnv("TEMPORAL_TASK_QUEUE", "nexus-task-queue"),
		// Rate limiting configuration (requests per minute)
		RateLimitOAuth:  getEnvInt("RATE_LIMIT_OAUTH", 5),   // OAuth endpoints (per IP)
		RateLimitHealth: getEnvInt("RATE_LIMIT_HEALTH", 60), // Health check (per IP)
		RateLimitAPI:    getEnvInt("RATE_LIMIT_API", 300),   // Authenticated API (per user)
		RateLimitAdmin:  getEnvInt("RATE_LIMIT_ADMIN", 100), // Admin endpoints (per user)
	}

	server, err := rest.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		log.Println("Starting REST API Gateway...")
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

func getAllowedOrigins() []string {
	// In development, allow localhost:3000 by default
	// In production, set via ALLOWED_ORIGINS env var (comma-separated)
	origins := getEnv("ALLOWED_ORIGINS", "http://localhost:3000")
	if origins == "*" {
		return []string{"*"}
	}

	result := []string{}
	for _, origin := range splitByComma(origins) {
		if origin != "" {
			result = append(result, origin)
		}
	}
	return result
}

func splitByComma(s string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func getDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "admin")
	password := getEnv("POSTGRES_PASSWORD", "")
	dbname := getEnv("POSTGRES_DB", "nexus_db")
	sslmode := getEnv("POSTGRES_SSLMODE", "disable")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode,
	)
}
