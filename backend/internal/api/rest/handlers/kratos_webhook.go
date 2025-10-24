// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/uuid"

	clientgraphql "github.com/retran/nexus/backend/internal/client/graphql"
)

// KratosWebhookHandlers handles webhooks from Kratos
type KratosWebhookHandlers struct {
	graphqlClient graphql.Client
	webhookSecret string
}

// NewKratosWebhookHandlers creates a new Kratos webhook handlers instance
func NewKratosWebhookHandlers(graphqlClient graphql.Client) *KratosWebhookHandlers {
	return &KratosWebhookHandlers{
		graphqlClient: graphqlClient,
		webhookSecret: os.Getenv("KRATOS_WEBHOOK_SECRET"),
	}
}

// KratosWebhookPayload represents the webhook payload from Kratos
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

// HandleRegistration handles user registration webhook from Kratos
func (h *KratosWebhookHandlers) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	// Verify webhook secret
	webhookSecret := r.Header.Get("X-Webhook-Secret")
	if h.webhookSecret != "" && webhookSecret != h.webhookSecret {
		http.Error(w, "Unauthorized: Invalid webhook secret", http.StatusUnauthorized)
		return
	}

	// Parse webhook payload
	var payload KratosWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if payload.IdentityID == "" || payload.Email == "" {
		http.Error(w, "Bad Request: identity_id and email are required", http.StatusBadRequest)
		return
	}

	// Parse Kratos identity ID as UUID
	kratosIdentityID, err := uuid.Parse(payload.IdentityID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: Invalid identity_id format: %v", err), http.StatusBadRequest)
		return
	}

	// Build display name
	var name *string
	if payload.Name.First != "" || payload.Name.Last != "" {
		displayName := fmt.Sprintf("%s %s", payload.Name.First, payload.Name.Last)
		displayName = trimSpace(displayName)
		if displayName != "" {
			name = &displayName
		}
	}

	// Build picture URL
	var picture *string
	if payload.Picture != "" {
		picture = &payload.Picture
	}

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

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
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
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "created",
		"user_id": createResp.CreateUser.Id,
		"message": "User created successfully with role=none (pending admin approval)",
	})
}

// trimSpace trims leading and trailing whitespace from a string
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
