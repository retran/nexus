// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package audit provides shared utilities for audit event handling.
package audit

import (
	"net/http"
	"strings"
)

// ExtractIPAddress extracts the client's IP address from the request.
// Checks X-Forwarded-For, X-Real-IP headers, falls back to RemoteAddr.
func ExtractIPAddress(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(forwarded, ","); idx > 0 {
			return strings.TrimSpace(forwarded[:idx])
		}
		return strings.TrimSpace(forwarded)
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	return r.RemoteAddr
}

// SanitizeID converts an event type or identifier into a safe workflow ID component.
// Trims, lowercases, replaces spaces with dashes, and limits length to 32 chars.
func SanitizeID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "event"
	}
	const maxLen = 32
	if len(value) > maxLen {
		value = value[:maxLen]
	}
	return strings.ReplaceAll(strings.ToLower(value), " ", "-")
}

// DefaultString returns value if non-empty (after trimming), otherwise returns fallback.
func DefaultString(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}
