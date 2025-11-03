// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package handlers contains HTTP handlers for REST API endpoints.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/retran/nexus/backend/internal/api/rest/middleware"
)

// AuthorizerHandler handles RBAC authorization requests from Oathkeeper.
type AuthorizerHandler struct {
	adminRoles []string
}

// NewAuthorizerHandler creates a new RBAC authorizer handler.
func NewAuthorizerHandler(adminRoles []string) *AuthorizerHandler {
	return &AuthorizerHandler{
		adminRoles: adminRoles,
	}
}

// AuthorizeRequest is the request payload from Oathkeeper.
type AuthorizeRequest struct {
	Context  map[string]string `json:"context"`
	Subject  string            `json:"subject"`
	Action   string            `json:"action"`
	Resource string            `json:"resource"`
}

// Authorize handles authorization requests from Oathkeeper remote_json authorizer.
// It checks if the authenticated user has the required role for the requested resource.
func (h *AuthorizerHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	authInfo := middleware.AuthInfoFromContext(r.Context())
	if authInfo == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// For now, implement simple admin check
	// In the future, this can be extended with more complex RBAC rules
	userRole := strings.ToLower(strings.TrimSpace(authInfo.Role))

	isAdmin := false
	for _, adminRole := range h.adminRoles {
		if strings.EqualFold(userRole, adminRole) {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// User is authorized
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"subject": authInfo.UserID.String(),
		"allowed": true,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
