// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package main bootstraps the Temporal worker responsible for audit processing.
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
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/retran/nexus/backend/internal/activities"
	gqlclient "github.com/retran/nexus/backend/internal/client/data"
	"github.com/retran/nexus/backend/internal/config"
	"github.com/retran/nexus/backend/internal/tracing"
	"github.com/retran/nexus/backend/internal/workflows"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Worker failed: %v", err)
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

	temporalHost := config.MustGetEnv("TEMPORAL_HOST")
	namespace := config.GetEnv("TEMPORAL_NAMESPACE", "default")
	taskQueue := config.MustGetEnv("TEMPORAL_TASK_QUEUE")

	log.Printf("Connecting to Temporal at %s...", temporalHost)

	c, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: namespace,
	})
	if err != nil {
		return fmt.Errorf("failed to create Temporal client: %w", err)
	}
	defer c.Close()
	log.Println("Connected to Temporal")

	dataAPIEndpoint := config.MustGetEnv("DATA_API_ENDPOINT")
	gqlClient := gqlclient.NewClient(dataAPIEndpoint)
	log.Printf("Initialized Data API client for: %s", dataAPIEndpoint)

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
