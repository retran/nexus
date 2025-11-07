// Package handlers provides internal API request handlers.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/retran/nexus/backend/internal/audit"
	"github.com/retran/nexus/backend/internal/domain"
)

// AuditHandler accepts audit events and forwards them to audit service.
type AuditHandler struct {
	service audit.Service
}

// NewAuditHandler constructs a handler for audit events.
func NewAuditHandler(auditService audit.Service) *AuditHandler {
	if auditService == nil {
		return nil
	}

	return &AuditHandler{
		service: auditService,
	}
}

// HandleAuditEvent accepts an audit event and triggers the audit workflow.
func (h *AuditHandler) HandleAuditEvent(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		http.Error(w, "Audit service not configured", http.StatusServiceUnavailable)
		return
	}

	event, err := decodeAuditEvent(r)
	if err != nil {
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.LogEvent(r.Context(), event); err != nil {
		http.Error(w, "Failed to record audit event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
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
		Source:    audit.DefaultString(payload.Source, "system"),
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
		event.IPAddress = audit.ExtractIPAddress(r)
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
