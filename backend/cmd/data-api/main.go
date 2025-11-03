// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the Data API (GraphQL) server executable.
// This service provides GraphQL API for database operations.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/retran/nexus/backend/internal/api/graphql"
	"github.com/retran/nexus/backend/internal/api/graphql/resolvers"
	"github.com/retran/nexus/backend/internal/auth"
	internalmiddleware "github.com/retran/nexus/backend/internal/internalapi/middleware"
	"github.com/retran/nexus/backend/internal/repository/postgres"
	"github.com/retran/nexus/backend/internal/secrets"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("data-api failed: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	dbConfig := &postgres.Config{
		Host:            getEnv("POSTGRES_HOST", "localhost"),
		Port:            5432,
		Database:        getEnv("POSTGRES_DB", "nexus"),
		User:            getEnv("POSTGRES_USER", "postgres"),
		Password:        getEnv("POSTGRES_PASSWORD", "postgres"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}

	log.Println("Connecting to database...")
	pool, err := postgres.NewPool(ctx, dbConfig)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	queries := postgres.New(pool)

	resolver := &resolvers.Resolver{
		Queries: queries,
	}

	secretsClient, jwtMiddleware, accessErr := setupAccessControl(ctx)
	if accessErr != nil {
		return fmt.Errorf("configure access control: %w", accessErr)
	}

	srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: resolver,
	}))

	mux := http.NewServeMux()
	mux.Handle("/graphql", protectHandler(secretsClient, jwtMiddleware, srv))
	mux.Handle("/", protectHandler(secretsClient, jwtMiddleware, playground.Handler("GraphQL Playground", "/graphql")))

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("health check response write failed: %v", err)
		}
	})

	port := getEnv("SERVER_PORT", "8081")
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("GraphQL server starting on http://localhost:%s/graphql", port)
		log.Printf("GraphQL Playground available at http://localhost:%s/", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	log.Println("Server stopped")
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupAccessControl(ctx context.Context) (*secrets.Client, *internalmiddleware.JWTMiddleware, error) {
	vaultAddr := getEnv("VAULT_ADDR", "http://vault:8200")
	roleID := strings.TrimSpace(getEnv("VAULT_ROLE_ID", ""))
	secretID := strings.TrimSpace(getEnv("VAULT_SECRET_ID", ""))
	authMount := getEnv("VAULT_AUTH_MOUNT_PATH", "approle")
	kvMount := getEnv("VAULT_KV_MOUNT_PATH", "kv")
	transitMount := getEnv("VAULT_TRANSIT_MOUNT_PATH", "transit")
	signingKey := strings.TrimSpace(getEnv("VAULT_SIGNING_KEY", "service-jwt-key"))
	allowedAud := parseCSV(getEnv("JWT_ALLOWED_AUDIENCES", "data-api"))
	if len(allowedAud) == 0 {
		allowedAud = []string{"data-api"}
	}

	if roleID == "" {
		return nil, nil, fmt.Errorf("VAULT_ROLE_ID is required")
	}
	if secretID == "" {
		return nil, nil, fmt.Errorf("VAULT_SECRET_ID is required")
	}
	if signingKey == "" {
		return nil, nil, fmt.Errorf("VAULT_SIGNING_KEY is required")
	}

	secretsClient, err := secrets.NewClient(&secrets.Config{
		Address:       vaultAddr,
		RoleID:        roleID,
		SecretID:      secretID,
		AuthMountPath: authMount,
		KVMountPath:   kvMount,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create Vault secrets client: %w", err)
	}

	vaultClient, err := secretsClient.APIClient(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("authenticate with Vault: %w", err)
	}

	verifier, err := auth.NewJWTVerifier(&auth.JWTVerifierConfig{
		VaultClient:      vaultClient,
		KeyName:          signingKey,
		TransitMountPath: transitMount,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("initialise JWT verifier: %w", err)
	}

	return secretsClient, internalmiddleware.NewJWTMiddleware(verifier, allowedAud), nil
}

func protectHandler(secretsClient *secrets.Client, jwtMiddleware *internalmiddleware.JWTMiddleware, next http.Handler) http.Handler {
	protected := jwtMiddleware.Require(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := secretsClient.APIClient(r.Context()); err != nil {
			log.Printf("failed to renew Vault token: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		protected.ServeHTTP(w, r)
	})
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
