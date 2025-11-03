// Package handlers provides internal API request handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/retran/nexus/backend/internal/domain"
	internalmiddleware "github.com/retran/nexus/backend/internal/internalapi/middleware"
)

// AuditHandler accepts audit events and forwards them to Temporal workflows.
type AuditHandler struct {
	client          client.Client
	taskQueue       string
	allowedSubjects []string
}

// NewAuditHandler constructs a handler for audit events.
func NewAuditHandler(temporalClient client.Client, taskQueue string, allowedSubjects []string) *AuditHandler {
	if temporalClient == nil {
		return nil
	}

	normalized := make([]string, 0, len(allowedSubjects))
	for _, subject := range allowedSubjects {
		if trimmed := strings.TrimSpace(subject); trimmed != "" {
			normalized = append(normalized, strings.ToLower(trimmed))
		}
	}
	return &AuditHandler{
		client:          temporalClient,
		taskQueue:       taskQueue,
		allowedSubjects: normalized,
	}
}

// HandleAuditEvent accepts an audit event and triggers the audit workflow.
func (h *AuditHandler) HandleAuditEvent(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		http.Error(w, "Audit service not configured", http.StatusServiceUnavailable)
		return
	}

	tokenInfo := internalmiddleware.TokenInfoFromContext(r.Context())
	if tokenInfo == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !h.subjectAllowed(tokenInfo.Subject) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	event, err := decodeAuditEvent(r)
	if err != nil {
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.dispatchWorkflow(r.Context(), event); err != nil {
		http.Error(w, "Failed to record audit event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *AuditHandler) subjectAllowed(subject string) bool {
	if len(h.allowedSubjects) == 0 {
		return true
	}
	subject = strings.ToLower(strings.TrimSpace(subject))
	for _, allowed := range h.allowedSubjects {
		if subject == allowed {
			return true
		}
	}
	return false
}

func (h *AuditHandler) dispatchWorkflow(ctx context.Context, event *domain.AuditEvent) error {
	if event == nil {
		return errors.New("audit event is required")
	}

	opts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("audit-%s-%s", sanitizeID(event.EventType), uuid.NewString()),
		TaskQueue: h.taskQueue,
	}

	_, err := h.client.ExecuteWorkflow(ctx, opts, "AuditLogWorkflow", *event)
	if err != nil {
		return fmt.Errorf("start audit workflow: %w", err)
	}
	return nil
}

func decodeAuditEvent(r *http.Request) (*domain.AuditEvent, error) {
	defer func() {
		if cerr := r.Body.Close(); cerr != nil {
			log.Printf("audit handler: close request body: %v", cerr)
		}
	}()

	var payload auditEventRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode body: %w", err)
	}

	if strings.TrimSpace(payload.EventType) == "" {
		return nil, errors.New("event_type is required")
	}

	event := domain.AuditEvent{
		EventType: payload.EventType,
		Metadata:  payload.Metadata,
		Source:    defaultString(payload.Source, "internal-api"),
		IPAddress: payload.IPAddress,
		UserAgent: payload.UserAgent,
	}

	if payload.UserID != "" {
		parsed, err := uuid.Parse(payload.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid user_id: %w", err)
		}
		event.UserID = &parsed
	}

	if payload.ResourceID != "" {
		event.ResourceID = &payload.ResourceID
	}

	if event.IPAddress == "" {
		event.IPAddress = r.Header.Get("X-Forwarded-For")
		if event.IPAddress == "" {
			event.IPAddress = r.RemoteAddr
		}
	}

	if event.UserAgent == "" {
		event.UserAgent = r.UserAgent()
	}

	return &event, nil
}

type auditEventRequest struct {
	Metadata   map[string]interface{} `json:"metadata"`
	UserID     string                 `json:"user_id"`
	ResourceID string                 `json:"resource_id"`
	EventType  string                 `json:"event_type"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	Source     string                 `json:"source"`
}

func defaultString(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func sanitizeID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "event"
	}
	const maxLen = 32
	if len(value) > maxLen {
		value = value[:maxLen]
	}
	return strings.ReplaceAll(strings.ToLower(value), " ", "-")
}
