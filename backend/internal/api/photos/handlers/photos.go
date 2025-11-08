// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP handlers for the photos API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// UnsplashPhoto represents the structure of a photo from Unsplash API.
type UnsplashPhoto struct {
	URLs struct {
		Regular string `json:"regular"`
	} `json:"urls"`
	User struct {
		Name  string `json:"name"`
		Links struct {
			HTML string `json:"html"`
		} `json:"links"`
	} `json:"user"`
}

// PhotoResponse represents the response for a single photo.
type PhotoResponse struct {
	ImageURL         string `json:"imageUrl"`
	PhotographerName string `json:"photographerName"`
	PhotographerLink string `json:"photographerLink"`
}

// PhotosResponse represents the response containing photos for both themes.
type PhotosResponse struct {
	Light PhotoResponse `json:"light"`
	Dark  PhotoResponse `json:"dark"`
}

// PhotosHandler handles photo-related HTTP requests.
type PhotosHandler struct {
	unsplashAccessKey string
}

// NewPhotosHandler creates a new PhotosHandler.
func NewPhotosHandler(accessKey string) *PhotosHandler {
	return &PhotosHandler{
		unsplashAccessKey: accessKey,
	}
}

// fetchRandomPhoto fetches a random photo from Unsplash based on the query.
// Falls back to picsum.photos if Unsplash is unavailable or access key is missing.
func fetchRandomPhoto(query string, accessKey string) *PhotoResponse {
	// If no access key, use fallback immediately
	if accessKey == "" {
		log.Printf("No Unsplash access key provided. Using picsum.photos fallback for query '%s'", query)
		return getFallbackPhoto()
	}

	// Try to fetch from Unsplash
	photo, err := fetchFromUnsplash(query, accessKey)
	if err != nil {
		log.Printf("Unsplash request failed for query '%s': %v. Falling back to picsum.photos", query, err)
		return getFallbackPhoto()
	}

	return photo
}

// fetchFromUnsplash attempts to fetch a photo from Unsplash API.
func fetchFromUnsplash(query, accessKey string) (*PhotoResponse, error) {
	// Build URL safely - only allow specific query values
	if query != "light" && query != "dark" {
		return nil, fmt.Errorf("invalid query: %s", query)
	}

	url := fmt.Sprintf("https://api.unsplash.com/photos/random?query=%s&client_id=%s", query, accessKey)

	// Create request with context for proper timeout and cancellation handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	// #nosec G107 - URL is constructed from validated query parameter and trusted access key
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch photo: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Warning: failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unsplash API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var photo UnsplashPhoto
	if err := json.Unmarshal(body, &photo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal photo: %w", err)
	}

	return &PhotoResponse{
		ImageURL:         photo.URLs.Regular,
		PhotographerName: photo.User.Name,
		PhotographerLink: photo.User.Links.HTML,
	}, nil
}

// getFallbackPhoto returns the fallback picsum.photos response.
func getFallbackPhoto() *PhotoResponse {
	return &PhotoResponse{
		ImageURL:         "https://picsum.photos/3840/2160",
		PhotographerName: "Picsum",
		PhotographerLink: "https://picsum.photos/",
	}
}

// GetPhotos handles GET /api/photos requests.
func (h *PhotosHandler) GetPhotos(w http.ResponseWriter, _ *http.Request) {
	// Fetch photo for light theme
	lightPhoto := fetchRandomPhoto("light", h.unsplashAccessKey)

	// Fetch photo for dark theme
	darkPhoto := fetchRandomPhoto("dark", h.unsplashAccessKey)

	response := PhotosResponse{
		Light: *lightPhoto,
		Dark:  *darkPhoto,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
