// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package services contains REST gateway service integrations.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/retran/nexus/backend/internal/domain"
)

const auditEndpointPath = "/internal/audit/events"

// TemporalAuditService sends audit events to the internal API.
type TemporalAuditService struct {
	httpClient *http.Client
	endpoint   string
}

// NewTemporalAuditService creates a new audit service that forwards events to the internal API.
func NewTemporalAuditService(httpClient *http.Client, baseURL string) *TemporalAuditService {
	base := strings.TrimSpace(baseURL)
	if httpClient == nil || base == "" {
		return nil
	}

	return &TemporalAuditService{
		httpClient: httpClient,
		endpoint:   strings.TrimRight(base, "/") + auditEndpointPath,
	}
}

// LogEvent sends an audit event asynchronously via the internal API.
func (s *TemporalAuditService) LogEvent(ctx context.Context, r *http.Request, userID *uuid.UUID, eventType string, metadata map[string]interface{}) error {
	if s == nil {
		return nil
	}

	event := s.buildEvent(r, userID, eventType, metadata)
	return s.postEvent(ctx, &event)
}

// LogEventSync sends an audit event synchronously, returning any error from the internal API.
func (s *TemporalAuditService) LogEventSync(ctx context.Context, r *http.Request, userID *uuid.UUID, eventType string, metadata map[string]interface{}) error {
	if s == nil {
		return nil
	}

	event := s.buildEvent(r, userID, eventType, metadata)
	return s.postEvent(ctx, &event)
}

func (s *TemporalAuditService) buildEvent(r *http.Request, userID *uuid.UUID, eventType string, metadata map[string]interface{}) domain.AuditEvent {
	return domain.AuditEvent{
		UserID:    userID,
		EventType: eventType,
		IPAddress: extractIPAddress(r),
		UserAgent: r.UserAgent(),
		Metadata:  metadata,
		Source:    "rest-gateway",
	}
}

func (s *TemporalAuditService) postEvent(ctx context.Context, event *domain.AuditEvent) error {
	if event == nil {
		return errors.New("audit event is required")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send audit request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("audit service: close response body: %v", cerr)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("internal api returned status %d", resp.StatusCode)
	}

	return nil
}

// extractIPAddress extracts the client's IP address from the request.
func extractIPAddress(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
