// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package handlers

import "net/http"

// UserHandlers handles user-related HTTP requests.
type UserHandlers struct{}

// NewUserHandlers creates a new UserHandlers instance.
func NewUserHandlers(_ interface{}) *UserHandlers {
	return &UserHandlers{}
}

// GetUser handles GET /api/users/:id.
func (h *UserHandlers) GetUser(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// GetUserByEmail handles GET /api/users/email/:email.
func (h *UserHandlers) GetUserByEmail(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// ListUsers handles GET /api/users.
func (h *UserHandlers) ListUsers(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// CreateUser handles POST /api/users.
func (h *UserHandlers) CreateUser(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// UpdateUser handles PUT /api/users/:id.
func (h *UserHandlers) UpdateUser(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// DeleteUser handles DELETE /api/users/:id.
func (h *UserHandlers) DeleteUser(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
