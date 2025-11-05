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
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/retran/nexus/backend/internal/api/data"
	"github.com/retran/nexus/backend/internal/repository/postgres"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("data failed: %v", err)
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

	resolver := &data.Resolver{
		Queries: queries,
	}

	srv := handler.NewDefaultServer(data.NewExecutableSchema(data.Config{
		Resolvers: resolver,
	}))

	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)
	mux.Handle("/", playground.Handler("GraphQL Playground", "/graphql"))

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
