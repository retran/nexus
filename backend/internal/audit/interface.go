// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package audit provides audit logging functionality.
package audit

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/retran/nexus/backend/internal/domain"
)

// Service defines the interface for audit logging.
type Service interface {
	// LogEvent records an audit event.
	LogEvent(ctx context.Context, event *domain.AuditEvent) error
}

// EventBuilder helps construct audit events from HTTP requests.
type EventBuilder struct {
	request  *http.Request
	metadata map[string]interface{}
	event    domain.AuditEvent
}

// NewEventBuilder creates a new audit event builder from an HTTP request.
func NewEventBuilder(r *http.Request) *EventBuilder {
	return &EventBuilder{
		request:  r,
		metadata: make(map[string]interface{}),
		event: domain.AuditEvent{
			IPAddress: ExtractIPAddress(r),
			UserAgent: r.UserAgent(),
		},
	}
}

// WithUserID sets the user ID for the audit event.
func (b *EventBuilder) WithUserID(userID *uuid.UUID) *EventBuilder {
	b.event.UserID = userID
	return b
}

// WithEventType sets the event type.
func (b *EventBuilder) WithEventType(eventType string) *EventBuilder {
	b.event.EventType = eventType
	return b
}

// WithSource sets the source of the audit event.
func (b *EventBuilder) WithSource(source string) *EventBuilder {
	b.event.Source = source
	return b
}

// WithResourceID sets the resource ID for the audit event.
func (b *EventBuilder) WithResourceID(resourceID string) *EventBuilder {
	b.event.ResourceID = &resourceID
	return b
}

// WithMetadata adds a metadata key-value pair.
func (b *EventBuilder) WithMetadata(key string, value interface{}) *EventBuilder {
	b.metadata[key] = value
	return b
}

// WithMetadataMap adds multiple metadata entries.
func (b *EventBuilder) WithMetadataMap(metadata map[string]interface{}) *EventBuilder {
	for k, v := range metadata {
		b.metadata[k] = v
	}
	return b
}

// Build constructs the final audit event.
func (b *EventBuilder) Build() domain.AuditEvent {
	if len(b.metadata) > 0 {
		b.event.Metadata = b.metadata
	}
	return b.event
}

// BuildAndLog constructs and logs the audit event using the provided service.
func (b *EventBuilder) BuildAndLog(ctx context.Context, service Service) error {
	if service == nil {
		return nil // No-op if service not configured
	}
	event := b.Build()
	if err := service.LogEvent(ctx, &event); err != nil {
		return fmt.Errorf("log audit event: %w", err)
	}
	return nil
}
