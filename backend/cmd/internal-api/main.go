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
	"strings"
	"syscall"
	"time"

	vaultapi "github.com/hashicorp/vault/api"

	"github.com/retran/nexus/backend/internal/auth"
	gql "github.com/retran/nexus/backend/internal/client/graphql"
	internalmiddleware "github.com/retran/nexus/backend/internal/internalapi/middleware"
	"github.com/retran/nexus/backend/internal/webhooks/kratos"
)

func main() {
	port := getEnv("PORT", "8083")
	graphqlEndpoint := getEnv("GRAPHQL_ENDPOINT", "http://localhost:8081/graphql")
	audienceEnv := getEnv("JWT_ALLOWED_AUDIENCES", "internal-api")

	log.Printf("GraphQL endpoint: %s", graphqlEndpoint)

	jwtVerifier, err := newJWTVerifierFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize JWT verifier: %v", err)
	}

	jwtMiddleware := internalmiddleware.NewJWTMiddleware(jwtVerifier, parseAudiences(audienceEnv))

	gqlClient := gql.NewClient(graphqlEndpoint)
	kratosHandler := kratos.NewHandler(gqlClient)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /webhooks/kratos/registration", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received Kratos webhook: method=%s path=%s from=%s", r.Method, r.URL.Path, r.RemoteAddr)
		kratosHandler.HandleRegistration(w, r)
	})

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /internal/healthz", internalHealthHandler)
	mux.Handle("/internal/", jwtMiddleware.Require(protectedMux))

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

func internalHealthHandler(w http.ResponseWriter, r *http.Request) {
	tokenInfo := internalmiddleware.TokenInfoFromContext(r.Context())
	response := "OK"
	if tokenInfo != nil && tokenInfo.Subject != "" {
		response = fmt.Sprintf("OK (%s)", tokenInfo.Subject)
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(response)); err != nil {
		log.Printf("Failed to write protected health response: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func newJWTVerifierFromEnv() (*auth.JWTVerifier, error) {
	vaultAddr := getEnv("VAULT_ADDR", "http://vault:8200")
	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		return nil, fmt.Errorf("VAULT_TOKEN environment variable is required")
	}

	signingKey := getEnv("VAULT_SIGNING_KEY", "service-jwt-key")
	transitPath := getEnv("VAULT_TRANSIT_MOUNT_PATH", "transit")

	cfg := vaultapi.DefaultConfig()
	cfg.Address = vaultAddr

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create Vault client: %w", err)
	}
	client.SetToken(vaultToken)

	verifier, err := auth.NewJWTVerifier(&auth.JWTVerifierConfig{
		VaultClient:      client,
		KeyName:          signingKey,
		TransitMountPath: transitPath,
	})
	if err != nil {
		return nil, fmt.Errorf("create JWT verifier: %w", err)
	}

	return verifier, nil
}

func parseAudiences(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
