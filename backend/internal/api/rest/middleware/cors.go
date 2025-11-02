// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package middleware

import "net/http"

// CORS adds CORS headers to responses.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := determineAllowedOrigin(r.Header.Get("Origin"), allowedOrigins); origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func determineAllowedOrigin(origin string, allowedOrigins []string) string {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" {
			if origin == "" {
				return "*"
			}
			return origin
		}
		if allowedOrigin == origin {
			return origin
		}
	}
	return ""
}
