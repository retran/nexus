// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package domain

import "github.com/google/uuid"

// AuditEvent represents an audit event to be logged.
type AuditEvent struct {
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	ResourceID *string                `json:"resource_id,omitempty"`
	EventType  string                 `json:"event_type"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Source     string                 `json:"source"`
}

// AuditEventType constants for common event types.
const (
	// Authentication events.
	AuditEventLoginSuccess = "login_success"
	AuditEventLoginFailed  = "login_failed"
	AuditEventLogout       = "logout"
	AuditEventTokenRefresh = "token_refresh"

	// User management events.
	AuditEventUserCreated     = "user_created"
	AuditEventUserUpdated     = "user_updated"
	AuditEventUserActivated   = "user_activated"
	AuditEventUserDeactivated = "user_deactivated"
	AuditEventRoleChanged     = "role_changed"
	AuditEventPasswordReset   = "password_reset"

	// Resource events (extensible for future use).
	AuditEventResourceCreated  = "resource_created"
	AuditEventResourceUpdated  = "resource_updated"
	AuditEventResourceDeleted  = "resource_deleted"
	AuditEventResourceAccessed = "resource_accessed"

	// System events.
	AuditEventSystemStartup  = "system_startup"
	AuditEventSystemShutdown = "system_shutdown"
	AuditEventConfigChanged  = "config_changed"
)
