// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the internal webhooks service.
// This service handles webhooks from external services (Kratos, etc.)
// and is NOT exposed to the public internet.
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

	gql "github.com/retran/nexus/backend/internal/client/graphql"
	"github.com/retran/nexus/backend/internal/webhooks/kratos"
)

func main() {
	port := getEnv("PORT", "8082")
	graphqlEndpoint := getEnv("GRAPHQL_ENDPOINT", "http://localhost:8081/graphql")

	log.Printf("Starting webhooks service on port %s", port)
	log.Printf("GraphQL endpoint: %s", graphqlEndpoint)

	// Initialize GraphQL client
	gqlClient := gql.NewClient(graphqlEndpoint)

	// Initialize handlers
	kratosHandler := kratos.NewHandler(gqlClient)

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Printf("Failed to write health check response: %v", err)
		}
	})

	// Kratos webhooks (internal only)
	mux.HandleFunc("POST /webhooks/kratos/registration", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received webhook request: method=%s path=%s from=%s", r.Method, r.URL.Path, r.RemoteAddr)
		kratosHandler.HandleRegistration(w, r)
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Webhooks service listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down webhooks service...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Webhooks service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
