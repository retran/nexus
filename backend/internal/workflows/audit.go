// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/retran/nexus/backend/internal/domain"
)

// AuditLogWorkflow processes audit events asynchronously.
func AuditLogWorkflow(ctx workflow.Context, event domain.AuditEvent) error {
	// Configure activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    5, // Retry up to 5 times
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Execute the audit logging activity
	err := workflow.ExecuteActivity(ctx, "RecordAuditLog", event).Get(ctx, nil)
	if err != nil {
		// Log failure but don't fail the workflow - audit is important but not critical
		workflow.GetLogger(ctx).Error("Failed to record audit log", "error", err, "event", event)
		return err
	}

	return nil
}

// BatchAuditLogWorkflow processes multiple audit events in a batch
// This is useful for high-volume scenarios.
func BatchAuditLogWorkflow(ctx workflow.Context, events []domain.AuditEvent) error {
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Execute batch recording
	err := workflow.ExecuteActivity(ctx, "RecordAuditLogBatch", events).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("Failed to record audit log batch", "error", err, "count", len(events))
		return err
	}

	return nil
}
