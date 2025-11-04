// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the internal API service. This service exposes
// internal-only endpoints for system-to-system communication.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

	"github.com/retran/nexus/backend/internal/internalapi/handlers"
	"github.com/retran/nexus/backend/internal/mtls"
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

	server, err := newHTTPServer(port, mux)
	if err != nil {
		return fmt.Errorf("create HTTP server: %w", err)
	}
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
	// mTLS ensures the request comes from an authorized service (gateway.service.local)
	response := "OK"
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		cn := r.TLS.PeerCertificates[0].Subject.CommonName
		response = fmt.Sprintf("OK (%s)", cn)
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

	// mTLS-protected endpoints (only gateway.service.local)
	mux.Handle("GET /internal/healthz", mTLSAuthMiddleware("gateway.service.local", http.HandlerFunc(internalHealthHandler)))
	mux.Handle("POST /admin/users/{id}/role", mTLSAuthMiddleware("gateway.service.local", adminRoleHandler(roleHandler)))
	if auditHandler != nil {
		mux.Handle("POST /internal/audit/events", mTLSAuthMiddleware("gateway.service.local", http.HandlerFunc(auditHandler.HandleAuditEvent)))
	}

	// Kratos webhooks (proxied via Oathkeeper with mTLS, CN = oathkeeper.service.local)
	if kratosWebhookHandler != nil {
		mux.Handle("POST /webhooks/kratos/registration", mTLSAuthMiddleware("oathkeeper.service.local", http.HandlerFunc(kratosWebhookHandler.HandleRegistration)))
		mux.Handle("POST /webhooks/kratos/login", mTLSAuthMiddleware("oathkeeper.service.local", http.HandlerFunc(kratosWebhookHandler.HandleLogin)))
		mux.Handle("POST /webhooks/kratos/logout", mTLSAuthMiddleware("oathkeeper.service.local", http.HandlerFunc(kratosWebhookHandler.HandleLogout)))
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
		// before calling this endpoint. mTLS ensures the request comes from gateway.service.local.

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

	// Load mTLS configuration for Temporal client
	tlsConfig, err := mtls.LoadClientTLSConfig("", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to load mTLS config: %w", err)
	}

	opts := temporalclient.Options{
		HostPort:  host,
		Namespace: strings.TrimSpace(namespace),
		ConnectionOptions: temporalclient.ConnectionOptions{
			TLS: tlsConfig,
		},
	}
	cli, err := temporalclient.Dial(opts)
	if err != nil {
		return nil, fmt.Errorf("connect to Temporal with mTLS: %w", err)
	}
	return cli, nil
}

func newHTTPServer(port string, handler http.Handler) (*http.Server, error) {
	tlsConfig, err := loadMTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load mTLS config: %w", err)
	}

	return &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		TLSConfig:    tlsConfig,
	}, nil
}

func serve(server *http.Server) error {
	errCh := make(chan error, 1)

	go func() {
		log.Printf("Internal API starting with mTLS on https://localhost%s", server.Addr)
		if err := server.ListenAndServeTLS("/secrets/tls.crt", "/secrets/tls.key"); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("listen and serve TLS: %w", err)
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

func mTLSAuthMiddleware(expectedCN string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check mTLS client certificate CN
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "Forbidden: missing client certificate", http.StatusForbidden)
			return
		}

		cn := r.TLS.PeerCertificates[0].Subject.CommonName
		if cn != expectedCN {
			log.Printf("Forbidden: invalid client CN: %s (expected %s)", cn, expectedCN)
			http.Error(w, "Forbidden: invalid client certificate", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
