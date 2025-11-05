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

	temporalclient "go.temporal.io/sdk/client"

	"github.com/retran/nexus/backend/internal/api/system/handlers"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Internal API terminated: %v", err)
	}
}

func run() error {
	port := getEnv("PORT", "8083")
	graphqlEndpoint := getEnv("GRAPHQL_ENDPOINT", "http://localhost:8081/graphql")
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

	roleHandler, auditHandler, kratosWebhookHandler, err := initServices(graphqlEndpoint, allowedRolesEnv, kratosAdminURL, temporalClient, taskQueue, auditSubjects)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	configureMux(mux, roleHandler, auditHandler, kratosWebhookHandler)

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

func internalHealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Failed to write protected health response: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
func configureMux(mux *http.ServeMux, roleHandler *handlers.AdminHandler, auditHandler *handlers.AuditHandler, kratosWebhookHandler *handlers.KratosWebhookHandler) {
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /internal/healthz", internalHealthHandler)
	mux.Handle("POST /admin/users/{id}/role", adminRoleHandler(roleHandler))

	if auditHandler != nil {
		mux.HandleFunc("POST /internal/audit/events", auditHandler.HandleAuditEvent)
	}

	if kratosWebhookHandler != nil {
		mux.HandleFunc("POST /webhooks/kratos/registration", kratosWebhookHandler.HandleRegistration)
		mux.HandleFunc("POST /webhooks/kratos/login", kratosWebhookHandler.HandleLogin)
		mux.HandleFunc("POST /webhooks/kratos/logout", kratosWebhookHandler.HandleLogout)
	}
}

func initServices(_ string, allowedRolesEnv, kratosAdminURL string, temporalClient temporalclient.Client, taskQueue, auditSubjects string) (*handlers.AdminHandler, *handlers.AuditHandler, *handlers.KratosWebhookHandler, error) {
	webhookSecret := getEnv("KRATOS_WEBHOOK_SECRET", "")
	if webhookSecret == "" {
		log.Println("Warning: KRATOS_WEBHOOK_SECRET not set, webhook validation disabled")
	}

	roleHandler, err := handlers.NewAdminHandler(kratosAdminURL, parseCSV(allowedRolesEnv))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create admin handler: %w", err)
	}

	auditHandler := handlers.NewAuditHandler(temporalClient, taskQueue, parseCSV(auditSubjects))
	kratosWebhookHandler := handlers.NewKratosWebhookHandler(temporalClient, taskQueue, webhookSecret)

	return roleHandler, auditHandler, kratosWebhookHandler, nil
}

func adminRoleHandler(roleHandler *handlers.AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Role validation is now handled by the gateway, which already verifies admin role
		// before calling this endpoint.

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

	opts := temporalclient.Options{
		HostPort:  host,
		Namespace: strings.TrimSpace(namespace),
	}
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
		log.Printf("Internal API starting on http://localhost%s", server.Addr)
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
