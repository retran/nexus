// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main starts the REST API gateway executable.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/retran/nexus/backend/internal/api/gateway"
	"github.com/retran/nexus/backend/internal/config"
)

func main() {
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
		RedisPassword:   config.GetEnv("REDIS_PASSWORD", ""),
		RedisDB:         config.GetEnvInt("REDIS_DB", 0),
		FrontendURL:     config.MustGetEnv("FRONTEND_URL"),
		InternalAPIURL:  config.MustGetEnv("SYSTEM_API_ENDPOINT"),
		KratosAdminURL:  config.MustGetEnv("KRATOS_ADMIN_URL"),
	}

	server, err := gateway.New(&cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		log.Println("Starting REST API Gateway...")
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
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
