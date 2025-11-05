// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP handlers for internal API endpoints.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/retran/nexus/backend/internal/domain"
)

// KratosWebhookHandler handles webhook events from Ory Kratos.
type KratosWebhookHandler struct {
	temporalClient temporalclient.Client
	taskQueue      string
	webhookSecret  string
}

// NewKratosWebhookHandler creates a new Kratos webhook handler.
func NewKratosWebhookHandler(temporalClient temporalclient.Client, taskQueue, webhookSecret string) *KratosWebhookHandler {
	return &KratosWebhookHandler{
		temporalClient: temporalClient,
		taskQueue:      taskQueue,
		webhookSecret:  webhookSecret,
	}
}

// KratosWebhookPayload represents the payload from Kratos webhooks.
type KratosWebhookPayload struct {
	Identity struct {
		ID     string `json:"id"`
		Traits struct {
			Email string `json:"email"`
			Name  struct {
				First string `json:"first"`
				Last  string `json:"last"`
			} `json:"name"`
			Role string `json:"role"`
		} `json:"traits"`
	} `json:"identity"`
	Flow struct {
		Type string `json:"type"` // "login" or "registration"
		ID   string `json:"id"`
	} `json:"flow"`
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
	if h.temporalClient == nil {
		return errors.New("temporal client not configured")
	}

	workflowID := fmt.Sprintf("audit-%s-%s-%d", eventType, payload.Identity.ID, time.Now().Unix())

	workflowOptions := temporalclient.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: h.taskQueue,
	}

	// Parse Kratos identity ID to UUID
	identityID, err := uuid.Parse(payload.Identity.ID)
	if err != nil {
		return fmt.Errorf("parse identity ID: %w", err)
	}

	// Create domain.AuditEvent to match the workflow signature
	event := domain.AuditEvent{
		UserID:    &identityID,
		EventType: eventType,
		Source:    "kratos",
		Metadata: map[string]interface{}{
			"identity_id": payload.Identity.ID,
			"email":       payload.Identity.Traits.Email,
			"role":        payload.Identity.Traits.Role,
			"flow_id":     payload.Flow.ID,
			"flow_type":   payload.Flow.Type,
		},
	}

	// Use the correct workflow name and type
	_, err = h.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "AuditLogWorkflow", event)
	if err != nil {
		return fmt.Errorf("execute audit workflow: %w", err)
	}

	return nil
}
