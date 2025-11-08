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
	"net/url"
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

// photoParams holds the parsed and validated query parameters.
type photoParams struct {
	collections   string
	topics        string
	username      string
	orientation   string
	contentFilter string
	query         string
}

// parseAndValidateParams extracts and validates query parameters.
func parseAndValidateParams(q url.Values) (*photoParams, error) {
	params := &photoParams{
		collections:   q.Get("collections"),
		topics:        q.Get("topics"),
		username:      q.Get("username"),
		orientation:   q.Get("orientation"),
		contentFilter: q.Get("content_filter"),
		query:         q.Get("query"),
	}

	if params.contentFilter == "" {
		params.contentFilter = "low"
	}

	// Validate incompatible params
	if (params.collections != "" || params.topics != "") && params.query != "" {
		return nil, fmt.Errorf("collections/topics cannot be combined with query")
	}

	// Validate orientation
	if params.orientation != "" {
		switch params.orientation {
		case "landscape", "portrait", "squarish":
			// ok
		default:
			return nil, fmt.Errorf("invalid orientation")
		}
	}

	return params, nil
}

// buildURLValues builds url.Values for a theme with the given query.
func (p *photoParams) buildURLValues(themeQuery string) url.Values {
	vals := url.Values{}
	if p.collections != "" {
		vals.Set("collections", p.collections)
	}
	if p.topics != "" {
		vals.Set("topics", p.topics)
	}
	if p.username != "" {
		vals.Set("username", p.username)
	}
	if p.query != "" {
		vals.Set("query", p.query)
	} else {
		vals.Set("query", themeQuery)
	}
	if p.orientation != "" {
		vals.Set("orientation", p.orientation)
	}
	if p.contentFilter != "" {
		vals.Set("content_filter", p.contentFilter)
	}
	return vals
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

// getFallbackPhoto returns the fallback picsum.photos response with 4K resolution.
func getFallbackPhoto() *PhotoResponse {
	return &PhotoResponse{
		ImageURL:         "https://picsum.photos/3840/2160",
		PhotographerName: "Picsum",
		PhotographerLink: "https://picsum.photos/",
	}
}

// GetRandomPhoto handles GET /photos/random requests.
// Returns photos for light and dark themes in Amsterdam, with optional filters applied to both.
func (h *PhotosHandler) GetRandomPhoto(w http.ResponseWriter, r *http.Request) {
	params, err := parseAndValidateParams(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch photo for light theme - Amsterdam daytime/sunny scenes
	lightVals := params.buildURLValues("Amsterdam day")
	lightPhotos, err := fetchRandomWithFilters(lightVals, h.unsplashAccessKey, 1)
	if err != nil {
		log.Printf("Failed to fetch light theme photo: %v", err)
		http.Error(w, "Failed to fetch photos", http.StatusInternalServerError)
		return
	}

	// Fetch photo for dark theme - Amsterdam evening/night scenes
	darkVals := params.buildURLValues("Amsterdam night")
	darkPhotos, err := fetchRandomWithFilters(darkVals, h.unsplashAccessKey, 1)
	if err != nil {
		log.Printf("Failed to fetch dark theme photo: %v", err)
		http.Error(w, "Failed to fetch photos", http.StatusInternalServerError)
		return
	}

	response := PhotosResponse{
		Light: lightPhotos[0],
		Dark:  darkPhotos[0],
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getFallbackPhotos returns a slice of fallback photos.
func getFallbackPhotos(count int) []PhotoResponse {
	res := make([]PhotoResponse, count)
	fb := getFallbackPhoto()
	for i := 0; i < count; i++ {
		res[i] = *fb
	}
	return res
}

// fetchRandomWithFilters calls Unsplash /photos/random with the provided URL values.
// Returns a slice of PhotoResponse (length = count or 1 on success). Falls back to picsum if needed.
func fetchRandomWithFilters(vals url.Values, accessKey string, count int) ([]PhotoResponse, error) {
	// If no access key, return fallback(s)
	if accessKey == "" {
		return getFallbackPhotos(count), nil
	}

	// Build request URL
	base := "https://api.unsplash.com/photos/random"
	vals.Set("client_id", accessKey)
	// Prefer 4K crop when possible
	vals.Set("w", "3840")
	vals.Set("h", "2160")
	vals.Set("fit", "crop")
	if vals.Get("orientation") == "" {
		vals.Set("orientation", "landscape")
	}

	fullURL := base + "?" + vals.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 12 * time.Second}
	// #nosec G107 - URL is constructed from validated inputs and trusted access key
	resp, err := client.Do(req)
	if err != nil {
		// fallback
		return getFallbackPhotos(count), nil
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Warning: failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		// fallback
		return getFallbackPhotos(count), nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return parseUnsplashResponse(body)
}

// parseUnsplashResponse parses the JSON response from Unsplash.
func parseUnsplashResponse(body []byte) ([]PhotoResponse, error) {
	// Unsplash returns either an object or an array depending on count
	var single UnsplashPhoto
	var many []UnsplashPhoto

	// Try unmarshalling as array first
	if err := json.Unmarshal(body, &many); err == nil && len(many) > 0 {
		res := make([]PhotoResponse, len(many))
		for i, p := range many {
			res[i] = PhotoResponse{
				ImageURL:         p.URLs.Regular + "&w=3840&h=2160&fit=crop",
				PhotographerName: p.User.Name,
				PhotographerLink: p.User.Links.HTML,
			}
		}
		return res, nil
	}

	if err := json.Unmarshal(body, &single); err != nil {
		return nil, fmt.Errorf("failed to unmarshal unsplash response: %w", err)
	}

	return []PhotoResponse{{
		ImageURL:         single.URLs.Regular + "&w=3840&h=2160&fit=crop",
		PhotographerName: single.User.Name,
		PhotographerLink: single.User.Links.HTML,
	}}, nil
}
