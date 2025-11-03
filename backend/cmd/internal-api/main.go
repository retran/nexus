// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the internal API service. This service exposes
// internal-only endpoints for system-to-system communication.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/retran/nexus/backend/internal/auth"
	"github.com/retran/nexus/backend/internal/internalapi/handlers"
	internalmiddleware "github.com/retran/nexus/backend/internal/internalapi/middleware"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Internal API terminated: %v", err)
	}
}

func run() error {
	port := getEnv("PORT", "8083")
	graphqlEndpoint := getEnv("GRAPHQL_ENDPOINT", "http://localhost:8081/graphql")
	audienceEnv := getEnv("JWT_ALLOWED_AUDIENCES", "internal-api")
	allowedRolesEnv := getEnv("INTERNAL_ALLOWED_ROLES", "none,member,admin")
	kratosAdminURL := getEnv("KRATOS_ADMIN_URL", "http://kratos:4434")
	temporalHost := getEnv("TEMPORAL_HOST", "temporal:7233")
	temporalNamespace := getEnv("TEMPORAL_NAMESPACE", "default")
	taskQueue := getEnv("TEMPORAL_TASK_QUEUE", "nexus-task-queue")
	auditSubjects := getEnv("AUDIT_ALLOWED_SUBJECTS", "gateway")

	log.Printf("GraphQL endpoint: %s", graphqlEndpoint)

	temporalClient, err := newTemporalClient(temporalHost, temporalNamespace)
	if err != nil {
		return err
	}
	defer func() {
		if temporalClient != nil {
			temporalClient.Close()
		}
	}()

	jwtMiddleware, roleHandler, auditHandler, err := initServices(graphqlEndpoint, audienceEnv, allowedRolesEnv, kratosAdminURL, temporalClient, taskQueue, auditSubjects)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	configureMux(mux, jwtMiddleware, roleHandler, auditHandler)

	server := newHTTPServer(port, mux)
	log.Printf("Starting internal API on port %s", port)

	return serve(server)
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

func parseCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
func configureMux(mux *http.ServeMux, jwtMiddleware *internalmiddleware.JWTMiddleware, roleHandler *handlers.AdminHandler, auditHandler *handlers.AuditHandler) {
	mux.HandleFunc("GET /health", healthHandler)

	mux.Handle("GET /internal/healthz", jwtMiddleware.Require(http.HandlerFunc(internalHealthHandler)))
	mux.Handle("POST /admin/users/{id}/role", jwtMiddleware.Require(adminRoleHandler(roleHandler)))
	if auditHandler != nil {
		mux.Handle("POST /internal/audit/events", jwtMiddleware.Require(http.HandlerFunc(auditHandler.HandleAuditEvent)))
	}
}

func initServices(_ string, audienceEnv, allowedRolesEnv, kratosAdminURL string, temporalClient temporalclient.Client, taskQueue, auditSubjects string) (*internalmiddleware.JWTMiddleware, *handlers.AdminHandler, *handlers.AuditHandler, error) {
	jwtVerifier, err := newJWTVerifierFromEnv()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("initialise JWT verifier: %w", err)
	}

	jwtMiddleware := internalmiddleware.NewJWTMiddleware(jwtVerifier, parseCSV(audienceEnv))
	roleHandler := handlers.NewAdminHandler(kratosAdminURL, parseCSV(allowedRolesEnv))
	auditHandler := handlers.NewAuditHandler(temporalClient, taskQueue, parseCSV(auditSubjects))

	return jwtMiddleware, roleHandler, auditHandler, nil
}

func adminRoleHandler(roleHandler *handlers.AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenInfo := internalmiddleware.TokenInfoFromContext(r.Context())
		if tokenInfo == nil || !strings.EqualFold(tokenInfo.Role, "admin") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		identityID := strings.TrimSpace(r.PathValue("id"))
		if identityID == "" {
			http.Error(w, "Bad Request: missing identity id", http.StatusBadRequest)
			return
		}

		var payload handlers.UpdateRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
			return
		}

		if err := roleHandler.UpdateUserRole(r.Context(), identityID, payload); err != nil {
			if errors.Is(err, handlers.ErrUnknownRole) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, "Failed to update role: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func newTemporalClient(host, namespace string) (temporalclient.Client, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, errors.New("TEMPORAL_HOST is required")
	}

	opts := temporalclient.Options{HostPort: host, Namespace: strings.TrimSpace(namespace)}
	cli, err := temporalclient.Dial(opts)
	if err != nil {
		return nil, fmt.Errorf("connect to Temporal: %w", err)
	}
	return cli, nil
}

func newHTTPServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func serve(server *http.Server) error {
	errCh := make(chan error, 1)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-sigChan:
		log.Println("Shutting down internal API...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Println("Internal API stopped")
	return nil
}
