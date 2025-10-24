// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/uuid"

	gql "github.com/retran/nexus/backend/internal/client/graphql"
)

// UserHandlers handles user-related HTTP requests.
type UserHandlers struct {
	gqlClient graphql.Client
}

// NewUserHandlers creates a new UserHandlers instance.
func NewUserHandlers(gqlClient graphql.Client) *UserHandlers {
	return &UserHandlers{
		gqlClient: gqlClient,
	}
}

// GetUser handles GET /api/users/:id.
func (h *UserHandlers) GetUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Call GraphQL service
	resp, err := gql.GetUser(r.Context(), h.gqlClient, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp.User == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.User)
}

// GetUserByEmail handles GET /api/users/email/:email.
func (h *UserHandlers) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.PathValue("email")
	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	resp, err := gql.GetUserByEmail(r.Context(), h.gqlClient, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp.UserByEmail == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.UserByEmail)
}

// ListUsers handles GET /api/users.
func (h *UserHandlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	limit := 50
	offset := 0

	if l := query.Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := query.Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	resp, err := gql.ListUsers(r.Context(), h.gqlClient, &limit, &offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.Users)
}

// CreateUser handles POST /api/users.
func (h *UserHandlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input gql.CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := gql.CreateUser(r.Context(), h.gqlClient, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp.CreateUser)
}

// UpdateUser handles PUT /api/users/:id.
func (h *UserHandlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var input gql.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := gql.UpdateUser(r.Context(), h.gqlClient, id, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.UpdateUser)
}

// DeleteUser handles DELETE /api/users/:id.
func (h *UserHandlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	_, err = gql.DeleteUser(r.Context(), h.gqlClient, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
