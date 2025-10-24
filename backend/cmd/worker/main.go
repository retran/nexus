// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/retran/nexus/backend/internal/activities"
	gqlclient "github.com/retran/nexus/backend/internal/client/graphql"
	"github.com/retran/nexus/backend/internal/workflows"
)

func main() {
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")
	namespace := getEnv("TEMPORAL_NAMESPACE", "default")
	taskQueue := getEnv("TEMPORAL_TASK_QUEUE", "nexus-task-queue")

	log.Printf("Connecting to Temporal at %s...", temporalHost)
	c, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()
	log.Println("Connected to Temporal")

	apiURL := getEnv("API_URL", "http://localhost:8081/graphql")
	gqlClient := gqlclient.NewClient(apiURL)
	log.Printf("Initialized GraphQL client for: %s", apiURL)

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
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
