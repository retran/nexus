// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/retran/nexus/backend/internal/domain"
)

// Client sends audit events to a System API endpoint via HTTP.
type Client struct {
	httpClient *http.Client
	endpoint   string
}

// NewClient creates a new HTTP-based audit client.
func NewClient(httpClient *http.Client, baseURL string) *Client {
	base := strings.TrimSpace(baseURL)
	if httpClient == nil || base == "" {
		return nil
	}

	return &Client{
		httpClient: httpClient,
		endpoint:   strings.TrimRight(base, "/") + "/internal/audit/events",
	}
}

// LogEvent sends an audit event via HTTP POST to the System API.
func (s *Client) LogEvent(ctx context.Context, event *domain.AuditEvent) error {
	if s == nil {
		return nil // No-op if service not configured
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

// Ensure Client implements Service interface.
var _ Service = (*Client)(nil)
