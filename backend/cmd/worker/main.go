// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main bootstraps the Temporal worker responsible for audit processing.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/retran/nexus/backend/internal/activities"
	gqlclient "github.com/retran/nexus/backend/internal/client/data"
	"github.com/retran/nexus/backend/internal/client/mtls"
	"github.com/retran/nexus/backend/internal/workflows"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

func run() error {
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")
	namespace := getEnv("TEMPORAL_NAMESPACE", "default")
	taskQueue := getEnv("TEMPORAL_TASK_QUEUE", "nexus-task-queue")

	log.Printf("Connecting to Temporal at %s with mTLS...", temporalHost)

	// Load mTLS configuration for Temporal client
	tlsConfig, err := mtls.LoadClientTLSConfig("", "", "")
	if err != nil {
		return fmt.Errorf("failed to load mTLS config: %w", err)
	}

	c, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: namespace,
		ConnectionOptions: client.ConnectionOptions{
			TLS: tlsConfig,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create Temporal client: %w", err)
	}
	defer c.Close()
	log.Println("Connected to Temporal with mTLS")

	apiURL := getEnv("API_URL", "https://data.service.local:8081/graphql")
	gqlClient, err := gqlclient.NewMTLSClient(apiURL)
	if err != nil {
		return fmt.Errorf("failed to create GraphQL client with mTLS: %w", err)
	}
	log.Printf("Initialized GraphQL client with mTLS for: %s", apiURL)

	w := worker.New(c, taskQueue, worker.Options{})

	w.RegisterWorkflow(workflows.AuditLogWorkflow)
	w.RegisterWorkflow(workflows.BatchAuditLogWorkflow)

	auditActivities := activities.NewAuditActivities(gqlClient)
	w.RegisterActivity(auditActivities.RecordAuditLog)
	w.RegisterActivity(auditActivities.RecordAuditLogBatch)

	log.Println("Registered audit workflows and activities")

	log.Printf("Starting worker on task queue: %s", taskQueue)

	go func() {
		err = w.Run(worker.InterruptCh())
		if err != nil {
			log.Fatalf("Worker error: %v", err)
		}
	}()

	log.Println("Worker started successfully")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")
	w.Stop()
	log.Println("Worker stopped")
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
