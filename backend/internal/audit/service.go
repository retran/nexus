// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/retran/nexus/backend/internal/domain"
)

// TemporalService sends audit events to Temporal workflows.
type TemporalService struct {
	client    client.Client
	taskQueue string
}

// NewTemporalService creates a new Temporal-based audit service.
func NewTemporalService(temporalClient client.Client, taskQueue string) *TemporalService {
	if temporalClient == nil {
		return nil
	}

	return &TemporalService{
		client:    temporalClient,
		taskQueue: taskQueue,
	}
}

// LogEvent starts an AuditLogWorkflow in Temporal.
func (s *TemporalService) LogEvent(ctx context.Context, event *domain.AuditEvent) error {
	if s == nil {
		return nil // No-op if service not configured
	}

	if event.EventType == "" {
		return errors.New("event_type is required")
	}

	opts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("audit-%s-%s", SanitizeID(event.EventType), uuid.NewString()),
		TaskQueue: s.taskQueue,
	}

	_, err := s.client.ExecuteWorkflow(ctx, opts, "AuditLogWorkflow", event)
	if err != nil {
		return fmt.Errorf("start audit workflow: %w", err)
	}
	return nil
}

// Ensure TemporalService implements Service interface.
var _ Service = (*TemporalService)(nil)
