// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the Data API (GraphQL) server executable.
// This service provides GraphQL API for database operations.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/retran/nexus/backend/internal/api/graphql"
	"github.com/retran/nexus/backend/internal/api/graphql/resolvers"
	"github.com/retran/nexus/backend/internal/repository/postgres"
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

	srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: resolver,
	}))

	mux := http.NewServeMux()
	mux.Handle("/graphql", mTLSAuthMiddleware(srv))
	mux.Handle("/", mTLSAuthMiddleware(playground.Handler("GraphQL Playground", "/graphql")))

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("health check response write failed: %v", err)
		}
	})

	port := getEnv("SERVER_PORT", "8081")

	tlsConfig, err := loadMTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to load mTLS config: %w", err)
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		TLSConfig:    tlsConfig,
	}

	go func() {
		log.Printf("GraphQL server starting with mTLS on https://localhost:%s/graphql", port)
		log.Printf("GraphQL Playground available at https://localhost:%s/", port)
		if err := httpServer.ListenAndServeTLS("/secrets/tls.crt", "/secrets/tls.key"); err != nil && err != http.ErrServerClosed {
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

func loadMTLSConfig() (*tls.Config, error) {
	// Load CA certificate for validating client certificates
	caCert, err := os.ReadFile("/secrets/vault-ca.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
		MinVersion: tls.VersionTLS13,
	}, nil
}

func mTLSAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check mTLS client certificate CN
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "Forbidden: missing client certificate", http.StatusForbidden)
			return
		}

		cn := r.TLS.PeerCertificates[0].Subject.CommonName
		// Allow gateway.service.local or worker.service.local
		if cn != "gateway.service.local" && cn != "worker.service.local" {
			log.Printf("Forbidden: invalid client CN: %s", cn)
			http.Error(w, "Forbidden: invalid client certificate", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
