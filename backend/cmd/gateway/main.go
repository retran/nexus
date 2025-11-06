// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the REST API gateway executable.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/retran/nexus/backend/internal/api/gateway"
	"github.com/retran/nexus/backend/internal/config"
	"github.com/retran/nexus/backend/internal/tracing"
)

func main() {
	ctx := context.Background()

	// Load secrets from Vault (before defer to ensure it's created first)
	vaultClient, err := config.NewVaultClient()
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	redisPassword, err := vaultClient.GetRedisPassword(ctx)
	if err != nil {
		log.Fatalf("Failed to load Redis password from Vault: %v", err)
	}

	// Initialize OpenTelemetry tracing
	shutdown, err2 := tracing.InitTracerProvider(ctx)
	if err2 != nil {
		log.Printf("Warning: Failed to initialize tracing: %v", err2)
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

	cfg := gateway.Config{
		Port:            config.GetEnvInt("SERVER_PORT", 8080),
		Host:            config.GetEnv("SERVER_HOST", "0.0.0.0"),
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		GraphQLEndpoint: config.MustGetEnv("DATA_API_ENDPOINT"),
		AllowedOrigins:  config.MustGetEnvCSV("ALLOWED_ORIGINS"),
		RedisHost:       config.MustGetEnv("REDIS_HOST"),
		RedisPort:       config.MustGetEnvInt("REDIS_PORT"),
		RedisPassword:   redisPassword,
		RedisDB:         config.GetEnvInt("REDIS_DB", 0),
		FrontendURL:     config.MustGetEnv("FRONTEND_URL"),
		InternalAPIURL:  config.MustGetEnv("SYSTEM_API_ENDPOINT"),
		KratosAdminURL:  config.MustGetEnv("KRATOS_ADMIN_URL"),
	}

	server, err := gateway.New(&cfg)
	if err != nil {
		log.Printf("Failed to create server: %v", err)
		return
	}

	go func() {
		log.Println("Starting REST API Gateway...")
		if err := server.Start(); err != nil {
			log.Printf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
