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

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/retran/nexus/backend/internal/api/system/handlers"
	"github.com/retran/nexus/backend/internal/config"
	"github.com/retran/nexus/backend/internal/tracing"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Internal API terminated: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	// Initialize OpenTelemetry tracing
	shutdown, err := tracing.InitTracerProvider(ctx)
	if err != nil {
		log.Printf("Warning: Failed to initialize tracing: %v", err)
	} else {
		defer func() {
			if err := shutdown(ctx); err != nil {
				log.Printf("Error shutting down tracer: %v", err)
			}
		}()
	}

	// Start Prometheus metrics server on port 9091
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		metricsServer := &http.Server{
			Addr:              ":9091",
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		}
		log.Println("Metrics server listening on :9091")
		if err := metricsServer.ListenAndServe(); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// Load secrets from Vault
	vaultClient, err := config.NewVaultClient()
	if err != nil {
		return fmt.Errorf("failed to create Vault client: %w", err)
	}

	webhookSecret, err := vaultClient.GetWebhookSecret(ctx)
	if err != nil {
		return fmt.Errorf("failed to load webhook secret from Vault: %w", err)
	}

	port := config.GetEnv("SERVER_PORT", "8083")
	kratosAdminURL := config.MustGetEnv("KRATOS_ADMIN_URL")
	temporalHost := config.MustGetEnv("TEMPORAL_HOST")
	temporalNamespace := config.GetEnv("TEMPORAL_NAMESPACE", "default")
	taskQueue := config.MustGetEnv("TEMPORAL_TASK_QUEUE")

	temporalClient, err := newTemporalClient(temporalHost, temporalNamespace)
	if err != nil {
		return err
	}
	defer func() {
		if temporalClient != nil {
			temporalClient.Close()
		}
	}()

	roleHandler, auditHandler, kratosWebhookHandler, err := initServices(kratosAdminURL, temporalClient, taskQueue, webhookSecret)
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

func initServices(kratosAdminURL string, temporalClient temporalclient.Client, taskQueue, webhookSecret string) (*handlers.AdminHandler, *handlers.AuditHandler, *handlers.KratosWebhookHandler, error) {
	roleHandler, err := handlers.NewAdminHandler(kratosAdminURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create admin handler: %w", err)
	}

	auditHandler := handlers.NewAuditHandler(temporalClient, taskQueue)
	kratosWebhookHandler := handlers.NewKratosWebhookHandler(temporalClient, taskQueue, webhookSecret)

	return roleHandler, auditHandler, kratosWebhookHandler, nil
}

func adminRoleHandler(roleHandler *handlers.AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	// Wrap handler with OpenTelemetry middleware for distributed tracing
	wrappedHandler := otelhttp.NewHandler(handler, "nexus-system")

	return &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      wrappedHandler,
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
