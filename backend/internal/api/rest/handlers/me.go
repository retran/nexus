// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package handlers contains HTTP handlers for REST API endpoints.
package handlers

import (
	"net/http"

	"github.com/retran/nexus/backend/internal/api/rest/services"
)

// MeHandlers handles /me endpoint for current user info.
type MeHandlers struct{}

// NewMeHandlers creates new me handlers.
func NewMeHandlers(_ *services.TemporalAuditService) *MeHandlers {
	return &MeHandlers{}
}

// GetMe returns the current authenticated user's information.
func (h *MeHandlers) GetMe(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// Logout clears the authentication cookie.
func (h *MeHandlers) Logout(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// GetToken returns the JWT token as JSON (for API clients that can't use cookies).
func (h *MeHandlers) GetToken(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
