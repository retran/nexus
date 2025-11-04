// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package handlers contains HTTP handlers for REST API endpoints.
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/retran/nexus/backend/internal/api/common/middleware"
	"github.com/retran/nexus/backend/internal/api/gateway/services"
	"github.com/retran/nexus/backend/internal/client/mtls"
)

// MeHandlers handles /me endpoint for current user info.
type MeHandlers struct {
	auditService   *services.TemporalAuditService
	kratosAdminURL string
}

// NewMeHandlers creates new me handlers.
func NewMeHandlers(auditService *services.TemporalAuditService, kratosAdminURL string) *MeHandlers {
	return &MeHandlers{
		kratosAdminURL: kratosAdminURL,
		auditService:   auditService,
	}
}

// GetMe returns the current authenticated user's information.
func (h *MeHandlers) GetMe(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// Logout revokes the Kratos session and redirects to the login page.
func (h *MeHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	authInfo := middleware.AuthInfoFromContext(r.Context())
	if authInfo == nil || authInfo.SessionID == "" {
		// No session to revoke, just return success
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Call Kratos Admin API to revoke the session
	sessionID := strings.TrimSpace(authInfo.SessionID)
	kratosURL := strings.TrimRight(h.kratosAdminURL, "/")
	deleteURL := fmt.Sprintf("%s/admin/sessions/%s", kratosURL, sessionID)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodDelete, deleteURL, http.NoBody)
	if err != nil {
		log.Printf("Failed to create Kratos session delete request: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	client, err := mtls.NewTLSHTTPClient()
	if err != nil {
		log.Printf("Failed to create TLS HTTP client: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to revoke Kratos session: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		log.Printf("Kratos session revoke returned unexpected status: %d", resp.StatusCode)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Log logout event to audit trail
	if h.auditService != nil {
		metadata := map[string]interface{}{
			"identity_id": authInfo.UserID.String(),
			"email":       authInfo.Email,
			"session_id":  authInfo.SessionID,
		}

		if err := h.auditService.LogEvent(r.Context(), r, &authInfo.UserID, "user.logout", metadata); err != nil {
			log.Printf("Failed to log logout audit event: %v", err)
			// Don't fail the logout if audit logging fails
		}
	}

	// Session revoked successfully
	w.WriteHeader(http.StatusNoContent)
}

// GetToken returns the JWT token as JSON (for API clients that can't use cookies).
func (h *MeHandlers) GetToken(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
