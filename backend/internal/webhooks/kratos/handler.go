// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package kratos provides webhook handlers for Ory Kratos identity events.
package kratos

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

// Handler handles webhooks from Kratos.
type Handler struct {
	graphqlClient graphql.Client
	webhookSecret string
}

// NewHandler creates a new Kratos webhook handler instance.
func NewHandler(graphqlClient graphql.Client) *Handler {
	return &Handler{
		graphqlClient: graphqlClient,
		webhookSecret: os.Getenv("KRATOS_WEBHOOK_SECRET"),
	}
}

// Payload represents the webhook payload from Kratos.
type Payload struct {
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
func (h *Handler) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.isWebhookSecretValid(r) {
		fmt.Printf("ERROR: Invalid webhook secret\n")
		http.Error(w, "Unauthorized: Invalid webhook secret", http.StatusUnauthorized)
		return
	}

	payload, err := parsePayload(r)
	if err != nil {
		fmt.Printf("ERROR: Failed to parse payload: %v\n", err)
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	fmt.Printf("DEBUG: Received webhook payload: identity_id=%s, email=%s\n", payload.IdentityID, payload.Email)

	if err := validatePayload(&payload); err != nil {
		fmt.Printf("ERROR: Invalid payload: %v\n", err)
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	kratosIdentityID, err := uuid.Parse(payload.IdentityID)
	if err != nil {
		fmt.Printf("ERROR: Invalid identity_id format: %v\n", err)
		http.Error(w, fmt.Sprintf("Bad Request: Invalid identity_id format: %v", err), http.StatusBadRequest)
		return
	}

	name := buildDisplayName(payload.Name.First, payload.Name.Last)
	picture := optionalString(payload.Picture)

	// Upsert user in database with role="none" (pending approval)
	ctx := context.Background()

	fmt.Printf("DEBUG: Checking if user exists with Kratos ID: %s\n", kratosIdentityID)

	// Check if user already exists
	existingUserResp, err := clientgraphql.GetUserByKratosId(ctx, h.graphqlClient, kratosIdentityID)
	if err == nil && existingUserResp.UserByKratosId != nil {
		fmt.Printf("DEBUG: User exists, updating profile\n")
		// User already exists, update profile info only
		_, err = clientgraphql.UpdateUser(ctx, h.graphqlClient, existingUserResp.UserByKratosId.Id, clientgraphql.UpdateUserInput{
			Name:    name,
			Picture: picture,
		})
		if err != nil {
			fmt.Printf("ERROR: Failed to update user: %v\n", err)
			http.Error(w, fmt.Sprintf("Internal Server Error: Failed to update user: %v", err), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "updated",
			"user_id": existingUserResp.UserByKratosId.Id,
		})
		return
	}

	// User doesn't exist, create new user
	fmt.Printf("DEBUG: Creating new user with email: %s\n", payload.Email)
	createResp, err := clientgraphql.CreateUser(ctx, h.graphqlClient, clientgraphql.CreateUserInput{
		KratosIdentityId: kratosIdentityID,
		Email:            payload.Email,
		Name:             name,
		Picture:          picture,
	})
	if err != nil {
		fmt.Printf("ERROR: Failed to create user: %v\n", err)
		http.Error(w, fmt.Sprintf("Internal Server Error: Failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("SUCCESS: Created user with ID: %s\n", createResp.CreateUser.Id)

	// Return success response
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  "created",
		"user_id": createResp.CreateUser.Id,
		"message": "User created successfully",
	})
}

func (h *Handler) isWebhookSecretValid(r *http.Request) bool {
	if h.webhookSecret == "" {
		return true
	}
	return r.Header.Get("X-Webhook-Secret") == h.webhookSecret
}

func parsePayload(r *http.Request) (Payload, error) {
	var payload Payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return Payload{}, fmt.Errorf("decode webhook payload: %w", err)
	}
	return payload, nil
}

func validatePayload(payload *Payload) error {
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

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}
