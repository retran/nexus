// Package handlers exposes admin-only operations for the internal API.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// ErrUnknownRole indicates that the requested role is not permitted.
var ErrUnknownRole = errors.New("unknown role")

type identityPatch struct {
	Traits struct {
		Role string `json:"role"`
	} `json:"traits"`
}

// AdminHandler exposes admin-only operations for the internal API.
type AdminHandler struct {
	client      *http.Client
	kratosAdmin string
	allowed     []string
}

// NewAdminHandler creates a new handler for role management.
func NewAdminHandler(kratosAdminURL string, allowedRoles []string) (*AdminHandler, error) {
	return &AdminHandler{
		client:      &http.Client{},
		kratosAdmin: strings.TrimSuffix(kratosAdminURL, "/"),
		allowed:     allowedRoles,
	}, nil
}

// UpdateRoleRequest represents the payload accepted by UpdateUserRole.
type UpdateRoleRequest struct {
	Role string `json:"role"`
}

// UpdateUserRole synchronises the requested role with Kratos identity traits.
func (h *AdminHandler) UpdateUserRole(ctx context.Context, identityID string, req UpdateRoleRequest) error {
	role := strings.TrimSpace(req.Role)
	if !h.roleAllowed(role) {
		return fmt.Errorf("%w: %s", ErrUnknownRole, role)
	}

	patch := identityPatch{}
	patch.Traits.Role = role

	bodyBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("build kratos payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/admin/identities/%s", h.kratosAdmin, identityID)
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create kratos request: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	reqHTTP.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(reqHTTP)
	if err != nil {
		return fmt.Errorf("kratos request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("system: close kratos response body: %v", cerr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("kratos returned %d and body read failed: %w", resp.StatusCode, readErr)
		}
		return fmt.Errorf("kratos returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func (h *AdminHandler) roleAllowed(role string) bool {
	if len(h.allowed) == 0 {
		return true
	}
	for _, allowed := range h.allowed {
		if strings.EqualFold(role, allowed) {
			return true
		}
	}
	return false
}
