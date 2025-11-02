// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP handlers for the REST API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/uuid"

	clientgraphql "github.com/retran/nexus/backend/internal/client/graphql"
)

// KratosWebhookHandlers handles webhooks from Kratos.
type KratosWebhookHandlers struct {
	graphqlClient graphql.Client
	webhookSecret string
}

// NewKratosWebhookHandlers creates a new Kratos webhook handlers instance.
func NewKratosWebhookHandlers(graphqlClient graphql.Client) *KratosWebhookHandlers {
	return &KratosWebhookHandlers{
		graphqlClient: graphqlClient,
		webhookSecret: os.Getenv("KRATOS_WEBHOOK_SECRET"),
	}
}

// KratosWebhookPayload represents the webhook payload from Kratos.
type KratosWebhookPayload struct {
	IdentityID string `json:"identity_id"`
	Email      string `json:"email"`
	Name       struct {
		First string `json:"first"`
		Last  string `json:"last"`
	} `json:"name"`
	Picture        string `json:"picture"`
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
}

// HandleRegistration handles user registration webhook from Kratos.
func (h *KratosWebhookHandlers) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.isWebhookSecretValid(r) {
		http.Error(w, "Unauthorized: Invalid webhook secret", http.StatusUnauthorized)
		return
	}

	payload, err := parseKratosWebhookPayload(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	if err := validateKratosWebhookPayload(&payload); err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	kratosIdentityID, err := uuid.Parse(payload.IdentityID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: Invalid identity_id format: %v", err), http.StatusBadRequest)
		return
	}

	name := buildDisplayName(payload.Name.First, payload.Name.Last)
	picture := optionalString(payload.Picture)

	// Upsert user in database with role="none" (pending approval)
	ctx := context.Background()

	// Check if user already exists
	existingUserResp, err := clientgraphql.GetUserByKratosId(ctx, h.graphqlClient, kratosIdentityID)
	if err == nil && existingUserResp.UserByKratosId != nil {
		// User already exists, update profile info only
		_, err = clientgraphql.UpdateUser(ctx, h.graphqlClient, existingUserResp.UserByKratosId.Id, clientgraphql.UpdateUserInput{
			Name:    name,
			Picture: picture,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Internal Server Error: Failed to update user: %v", err), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "updated",
			"user_id": existingUserResp.UserByKratosId.Id,
		})
		return
	}

	// User doesn't exist, create new user with role="none"
	role := clientgraphql.UserRoleNone
	createResp, err := clientgraphql.CreateUser(ctx, h.graphqlClient, clientgraphql.CreateUserInput{
		KratosIdentityId: kratosIdentityID,
		Email:            payload.Email,
		Name:             name,
		Picture:          picture,
		Role:             &role,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal Server Error: Failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  "created",
		"user_id": createResp.CreateUser.Id,
		"message": "User created successfully with role=none (pending admin approval)",
	})
}

func (h *KratosWebhookHandlers) isWebhookSecretValid(r *http.Request) bool {
	if h.webhookSecret == "" {
		return true
	}
	return r.Header.Get("X-Webhook-Secret") == h.webhookSecret
}

func parseKratosWebhookPayload(r *http.Request) (KratosWebhookPayload, error) {
	var payload KratosWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return KratosWebhookPayload{}, fmt.Errorf("decode webhook payload: %w", err)
	}
	return payload, nil
}

func validateKratosWebhookPayload(payload *KratosWebhookPayload) error {
	if payload == nil || payload.IdentityID == "" || payload.Email == "" {
		return fmt.Errorf("identity_id and email are required")
	}
	return nil
}

func buildDisplayName(first, last string) *string {
	full := strings.TrimSpace(fmt.Sprintf("%s %s", first, last))
	if full == "" {
		return nil
	}
	return &full
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	v := value
	return &v
}
