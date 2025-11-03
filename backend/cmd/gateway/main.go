// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the REST API gateway executable.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/retran/nexus/backend/internal/api/rest"
)

func main() {
	cfg := rest.Config{
		Port:                  8080,
		Host:                  "0.0.0.0",
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		ShutdownTimeout:       30 * time.Second,
		GraphQLEndpoint:       getEnv("GRAPHQL_ENDPOINT", "http://localhost:8081/graphql"),
		AllowedOrigins:        getAllowedOrigins(),
		DatabaseURL:           getDatabaseURL(),
		RedisHost:             getEnv("REDIS_HOST", "localhost"),
		RedisPort:             getEnvInt("REDIS_PORT", 6379),
		RedisPassword:         getEnv("REDIS_PASSWORD", ""),
		RedisDB:               getEnvInt("REDIS_DB", 0),
		GoogleClientID:        getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:    getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:     getEnv("GOOGLE_REDIRECT_URL", "http://api.nexus.local/api/auth/google/callback"),
		FrontendURL:           getEnv("FRONTEND_URL", "http://nexus.local"),
		VaultAddress:          getEnv("VAULT_ADDR", "http://vault:8200"),
		VaultRoleID:           getEnv("VAULT_ROLE_ID", ""),
		VaultSecretID:         getEnv("VAULT_SECRET_ID", ""),
		VaultAuthMountPath:    getEnv("VAULT_AUTH_MOUNT_PATH", "approle"),
		VaultKVMountPath:      getEnv("VAULT_KV_MOUNT_PATH", "kv"),
		VaultTransitMountPath: getEnv("VAULT_TRANSIT_MOUNT_PATH", "transit"),
		VaultSigningKey:       getEnv("VAULT_SIGNING_KEY", "service-jwt-key"),
		ServiceJWTAudience:    parseCSVEnv("SERVICE_JWT_AUDIENCE", "data-api"),
		InternalAPIURL:        getEnv("INTERNAL_API_URL", "http://internal-api:8083"),
		InternalAPIAudience:   parseCSVEnv("INTERNAL_API_AUDIENCE", "internal-api"),
		ServiceJWTSubject:     getEnv("SERVICE_JWT_SUBJECT", "gateway"),
		ServiceJWTIssuer:      getEnv("SERVICE_JWT_ISSUER", "nexus"),
		ServiceJWTTTL:         getEnvDuration("SERVICE_JWT_TTL", 5*time.Minute),
		OathkeeperJWTIssuer:   getEnv("OATHKEEPER_JWT_ISSUER", "http://auth.nexus.local"),
		OathkeeperJWTAudience: getEnv("OATHKEEPER_JWT_AUDIENCE", "gateway"),
		OathkeeperJWKSFile:    getEnv("OATHKEEPER_JWKS_FILE", "/etc/oathkeeper/id_token.jwks.json"),
		// Rate limiting removed - now handled by Traefik at edge level
	}

	server, err := rest.New(&cfg)
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
		log.Printf("Server forced to shutdown: %v", err)
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

	return parseCSV(origins)
}

func parseCSVEnv(key, defaultValue string) []string {
	return parseCSV(getEnv(key, defaultValue))
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

func parseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
		log.Printf("Invalid duration for %s: %q, using default %s", key, value, defaultValue)
	}
	return defaultValue
}
