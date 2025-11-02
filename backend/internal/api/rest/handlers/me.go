// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"log"
	"net/http"

	"github.com/retran/nexus/backend/internal/api/rest/middleware"
	"github.com/retran/nexus/backend/internal/api/rest/services"
)

// MeHandlers handles /me endpoint for current user info.
type MeHandlers struct {
	auditService *services.TemporalAuditService
}

// NewMeHandlers creates new me handlers.
func NewMeHandlers(auditService *services.TemporalAuditService) *MeHandlers {
	return &MeHandlers{
		auditService: auditService,
	}
}

// GetMe returns the current authenticated user's information.
func (h *MeHandlers) GetMe(w http.ResponseWriter, r *http.Request) {
	// Get auth info from context (added by middleware)
	authInfo := middleware.GetAuthInfo(r.Context())
	if authInfo == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Return user info
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":   authInfo.UserID,
		"email":     authInfo.Email,
		"full_name": authInfo.FullName,
		"role":      authInfo.Role,
	})
}

// Logout clears the authentication cookie.
func (h *MeHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Get auth info before clearing
	authInfo := middleware.GetAuthInfo(r.Context())

	// Clear the auth cookie
	cookie := &http.Cookie{
		Name:     "nexus_auth",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete cookie
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)

	// Log logout event
	if authInfo != nil && h.auditService != nil {
		if err := h.auditService.LogEvent(r.Context(), r, &authInfo.UserID, "logout", nil); err != nil {
			log.Printf("failed to record logout audit event: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// GetToken returns the JWT token as JSON (for API clients that can't use cookies).
func (h *MeHandlers) GetToken(w http.ResponseWriter, r *http.Request) {
	// Get token from cookie
	cookie, err := r.Cookie("nexus_auth")
	if err != nil {
		http.Error(w, "No authentication token found", http.StatusUnauthorized)
		return
	}

	// Return token
	writeJSON(w, http.StatusOK, map[string]string{
		"token": cookie.Value,
		"type":  "Bearer",
	})
}
