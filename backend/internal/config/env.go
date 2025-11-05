// Copyright 2025 Andrew Vasilyev

// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration utilities.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MustGetEnv returns the value of the environment variable or panics if not set.
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

// MustGetEnvInt returns the integer value of the environment variable or panics if not set or invalid.
func MustGetEnvInt(key string) int {
	value := MustGetEnv(key)
	intValue, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("environment variable %s must be a valid integer, got: %s", key, value))
	}
	return intValue
}

// MustGetEnvBool returns the boolean value of the environment variable or panics if not set or invalid.
func MustGetEnvBool(key string) bool {
	value := MustGetEnv(key)
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("environment variable %s must be a valid boolean, got: %s", key, value))
	}
	return boolValue
}

// MustGetEnvCSV returns a slice of trimmed strings from a comma-separated environment variable.
// Panics if the environment variable is not set.
func MustGetEnvCSV(key string) []string {
	value := MustGetEnv(key)
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		panic(fmt.Sprintf("environment variable %s cannot be empty after parsing CSV", key))
	}
	return result
}

// GetEnv returns the value of the environment variable or the default value if not set.
// Use this only for truly optional configuration with sensible defaults.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt returns the integer value of the environment variable or the default value if not set.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
