// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/retran/nexus/backend/internal/domain"
)

// TemporalAuditService sends audit events to Temporal workflows.
type TemporalAuditService struct {
	temporalClient client.Client
	taskQueue      string
}

// NewTemporalAuditService creates a new Temporal-based audit service.
func NewTemporalAuditService(temporalClient client.Client, taskQueue string) *TemporalAuditService {
	return &TemporalAuditService{
		temporalClient: temporalClient,
		taskQueue:      taskQueue,
	}
}

// LogEvent sends an audit event to Temporal workflow asynchronously.
func (s *TemporalAuditService) LogEvent(ctx context.Context, r *http.Request, userID *uuid.UUID, eventType string, metadata map[string]interface{}) error {
	event := domain.AuditEvent{
		UserID:    userID,
		EventType: eventType,
		IPAddress: extractIPAddress(r),
		UserAgent: r.UserAgent(),
		Metadata:  metadata,
		Source:    "rest-gateway",
	}

	// Start workflow with unique ID based on timestamp and event type
	workflowID := fmt.Sprintf("audit-%s-%d", eventType, ctx.Value("request_id"))

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: s.taskQueue,
	}

	// Execute workflow asynchronously (fire and forget)
	_, err := s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "AuditLogWorkflow", event)
	if err != nil {
		// Log error but don't fail the request
		// Audit logging failure should not impact user experience
		return fmt.Errorf("failed to start audit workflow: %w", err)
	}

	return nil
}

// LogEventSync sends an audit event and waits for completion (use sparingly).
func (s *TemporalAuditService) LogEventSync(ctx context.Context, r *http.Request, userID *uuid.UUID, eventType string, metadata map[string]interface{}) error {
	event := domain.AuditEvent{
		UserID:    userID,
		EventType: eventType,
		IPAddress: extractIPAddress(r),
		UserAgent: r.UserAgent(),
		Metadata:  metadata,
		Source:    "rest-gateway",
	}

	workflowID := fmt.Sprintf("audit-%s-sync-%d", eventType, ctx.Value("request_id"))

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: s.taskQueue,
	}

	// Execute and wait for result
	run, err := s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "AuditLogWorkflow", event)
	if err != nil {
		return fmt.Errorf("failed to start audit workflow: %w", err)
	}

	// Wait for workflow completion
	err = run.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("audit workflow failed: %w", err)
	}

	return nil
}

// extractIPAddress extracts the client's IP address from the request.
func extractIPAddress(r *http.Request) string {
	// Check for X-Forwarded-For header (behind proxy)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	// Check for X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	// Fallback to RemoteAddr
	return r.RemoteAddr
}
