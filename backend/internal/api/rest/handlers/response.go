// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// writeJSON writes the JSON response and logs encoding errors.
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if status != 0 {
		w.WriteHeader(status)
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}
