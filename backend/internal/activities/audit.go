// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package activities

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/uuid"

	gqlclient "github.com/retran/nexus/backend/internal/client/graphql"
	"github.com/retran/nexus/backend/internal/domain"
)

// AuditActivities contains activities for audit logging.
type AuditActivities struct {
	gqlClient graphql.Client
}

// NewAuditActivities creates a new AuditActivities instance.
func NewAuditActivities(gqlClient graphql.Client) *AuditActivities {
	return &AuditActivities{
		gqlClient: gqlClient,
	}
}

// RecordAuditLog records a single audit event via GraphQL API.
func (a *AuditActivities) RecordAuditLog(ctx context.Context, event domain.AuditEvent) error {
	var metadataJSON *string
	if event.Metadata != nil {
		if event.Metadata == nil {
			event.Metadata = make(map[string]interface{})
		}
		event.Metadata["source"] = event.Source

		if event.ResourceID != nil {
			event.Metadata["resource_id"] = *event.ResourceID
		}

		bytes, err := json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		str := string(bytes)
		metadataJSON = &str
	}

	var userID *uuid.UUID
	if event.UserID != nil {
		userID = event.UserID
	}

	var ipAddress *string
	if event.IPAddress != "" {
		ipAddress = &event.IPAddress
	}

	var userAgent *string
	if event.UserAgent != "" {
		userAgent = &event.UserAgent
	}

	_, err := gqlclient.CreateAuditLog(ctx, a.gqlClient, gqlclient.CreateAuditLogInput{
		UserId:    userID,
		EventType: gqlclient.AuditEventType(event.EventType),
		IpAddress: ipAddress,
		UserAgent: userAgent,
		Metadata:  metadataJSON,
	})

	if err != nil {
		return fmt.Errorf("failed to create audit log via GraphQL: %w", err)
	}

	return nil
}

// RecordAuditLogBatch records multiple audit events in a single transaction.
func (a *AuditActivities) RecordAuditLogBatch(ctx context.Context, events []domain.AuditEvent) error {
	// For now, insert one by one
	// TODO: Implement batch insert for better performance
	for _, event := range events {
		if err := a.RecordAuditLog(ctx, event); err != nil {
			return fmt.Errorf("failed to record event in batch: %w", err)
		}
	}

	return nil
}
