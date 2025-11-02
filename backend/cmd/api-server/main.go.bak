// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/retran/nexus/backend/internal/api/graphql"
	"github.com/retran/nexus/backend/internal/api/graphql/resolvers"
	"github.com/retran/nexus/backend/internal/repository/postgres"
)

func main() {
	ctx := context.Background()

	dbConfig := postgres.Config{
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
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	queries := postgres.New(pool)

	resolver := &resolvers.Resolver{
		Queries: queries,
	}

	srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: resolver,
	}))

	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)
	mux.Handle("/", playground.Handler("GraphQL Playground", "/graphql"))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
