// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package handlers contains HTTP handlers for REST API endpoints.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/retran/nexus/backend/internal/api/common/middleware"
	"github.com/retran/nexus/backend/internal/audit"
)

// MeHandlers handles /me endpoint for current user info.
type MeHandlers struct {
	auditService audit.Service
}

// NewMeHandlers creates new me handlers.
func NewMeHandlers(auditService audit.Service) *MeHandlers {
	return &MeHandlers{
		auditService: auditService,
	}
}

// GetMe returns the current authenticated user's information.
func (h *MeHandlers) GetMe(w http.ResponseWriter, r *http.Request) {
	authInfo := middleware.AuthInfoFromContext(r.Context())
	if authInfo == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"id":        authInfo.UserID,
		"email":     authInfo.Email,
		"name":      authInfo.FullName,
		"role":      authInfo.Role,
		"sessionId": authInfo.SessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode /me response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
