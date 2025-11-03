// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the internal API service. This service exposes
// internal-only endpoints for system-to-system communication.
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
	port := getEnv("PORT", "8083")
	graphqlEndpoint := getEnv("GRAPHQL_ENDPOINT", "http://localhost:8081/graphql")

	log.Printf("GraphQL endpoint: %s", graphqlEndpoint)

	gqlClient := gql.NewClient(graphqlEndpoint)
	kratosHandler := kratos.NewHandler(gqlClient)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /webhooks/kratos/registration", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received Kratos webhook: method=%s path=%s from=%s", r.Method, r.URL.Path, r.RemoteAddr)
		kratosHandler.HandleRegistration(w, r)
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting internal API on port %s", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start internal API: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down internal API...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Internal API shutdown error: %v", err)
	}

	log.Println("Internal API stopped")
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Failed to write health response: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
