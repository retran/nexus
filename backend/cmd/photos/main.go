// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the Photos API service. This service provides
// public API endpoints for retrieving photos.
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

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/retran/nexus/backend/internal/api/photos/handlers"
	"github.com/retran/nexus/backend/internal/config"
	"github.com/retran/nexus/backend/internal/tracing"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Photos API terminated: %v", err)
	}
}

func run() error {
	ctx := context.Background()

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

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		metricsServer := &http.Server{
			Addr:              ":9096",
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		}
		log.Println("Metrics server listening on :9096")
		if err := metricsServer.ListenAndServe(); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	vaultClient, err := config.NewVaultClient()
	if err != nil {
		log.Printf("Warning: Failed to create Vault client: %v. Using empty access key.", err)
		vaultClient = nil
	}

	var unsplashAccessKey string
	if vaultClient != nil {
		unsplashAccessKey, err = vaultClient.GetUnsplashAccessKey(ctx)
		if err != nil {
			log.Printf("Warning: Failed to load Unsplash access key from Vault: %v. Using empty access key.", err)
			unsplashAccessKey = ""
		}
	}

	port := config.GetEnv("SERVER_PORT", "8084")

	photosHandler := handlers.NewPhotosHandler(unsplashAccessKey)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/photos", photosHandler.GetPhotos)

	server := newHTTPServer(port, mux)
	log.Printf("Starting Photos API on port %s", port)

	return serve(server)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Failed to write health response: %v", err)
	}
}

func newHTTPServer(port string, handler http.Handler) *http.Server {
	// Wrap handler with OpenTelemetry middleware for distributed tracing
	wrappedHandler := otelhttp.NewHandler(handler, "nexus-photos")

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
		log.Printf("Photos API starting on http://localhost%s", server.Addr)
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
		log.Println("Shutting down Photos API...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Println("Photos API stopped")
	return nil
}
