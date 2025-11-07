// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP handlers for internal API endpoints.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/retran/nexus/backend/internal/audit"
	"github.com/retran/nexus/backend/internal/domain"
)

// KratosWebhookHandler handles webhook events from Ory Kratos.
type KratosWebhookHandler struct {
	auditService  audit.Service
	webhookSecret string
}

// NewKratosWebhookHandler creates a new Kratos webhook handler.
func NewKratosWebhookHandler(auditService audit.Service, webhookSecret string) *KratosWebhookHandler {
	return &KratosWebhookHandler{
		auditService:  auditService,
		webhookSecret: webhookSecret,
	}
}

// KratosWebhookPayload represents the payload from Kratos webhooks.
type KratosWebhookPayload struct {
	IdentityID string `json:"identity_id"`
	Email      string `json:"email"`
	Name       struct {
		First string `json:"first"`
		Last  string `json:"last"`
	} `json:"name"`
	Picture  string `json:"picture"`
	FlowType string `json:"flow_type"`
	FlowID   string `json:"flow_id"`
}

// HandleRegistration handles user registration webhooks from Kratos.
func (h *KratosWebhookHandler) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, "user.registered", "registration")
}

// HandleLogin handles user login webhooks from Kratos.
func (h *KratosWebhookHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, "user.login", "login")
}

// HandleLogout handles user logout webhooks from Kratos.
func (h *KratosWebhookHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, "user.logout", "logout")
}

func (h *KratosWebhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request, eventType, eventName string) {
	if !h.validateWebhookSecret(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var payload KratosWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.logAuditEvent(r.Context(), eventType, &payload); err != nil {
		log.Printf("Failed to log %s audit event: %v", eventName, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *KratosWebhookHandler) validateWebhookSecret(r *http.Request) bool {
	secret := strings.TrimSpace(r.Header.Get("X-Webhook-Secret"))
	if secret == "" {
		log.Println("Missing X-Webhook-Secret header")
		return false
	}

	expected := strings.TrimSpace(h.webhookSecret)
	if expected == "" {
		log.Println("Webhook secret not configured")
		return false
	}

	if secret != expected {
		log.Println("Invalid webhook secret")
		return false
	}

	return true
}

func (h *KratosWebhookHandler) logAuditEvent(ctx context.Context, eventType string, payload *KratosWebhookPayload) error {
	if h.auditService == nil {
		return fmt.Errorf("audit service not configured")
	}

	identityID, err := uuid.Parse(payload.IdentityID)
	if err != nil {
		return fmt.Errorf("parse identity ID: %w", err)
	}

	event := domain.AuditEvent{
		UserID:    &identityID,
		EventType: eventType,
		Source:    "kratos",
		Metadata: map[string]interface{}{
			"identity_id": payload.IdentityID,
			"email":       payload.Email,
			"name":        payload.Name,
			"picture":     payload.Picture,
			"flow_id":     payload.FlowID,
			"flow_type":   payload.FlowType,
		},
	}

	if err := h.auditService.LogEvent(ctx, &event); err != nil {
		return fmt.Errorf("log audit event: %w", err)
	}

	return nil
}
