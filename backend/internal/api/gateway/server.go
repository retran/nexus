// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package gateway exposes configuration and startup helpers for the REST gateway.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/redis/go-redis/v9"

	"github.com/retran/nexus/backend/internal/api/common/middleware"
	"github.com/retran/nexus/backend/internal/api/gateway/handlers"
	"github.com/retran/nexus/backend/internal/api/gateway/services"
	gql "github.com/retran/nexus/backend/internal/client/data"
)

// Config contains REST API Gateway server configuration.
type Config struct {
	GraphQLEndpoint string
	RedisHost       string
	Host            string
	RedisPassword   string
	FrontendURL     string
	InternalAPIURL  string
	KratosAdminURL  string
	AllowedOrigins  []string
	ShutdownTimeout time.Duration
	WriteTimeout    time.Duration
	ReadTimeout     time.Duration
	Port            int
	RedisDB         int
	RedisPort       int
}

// Server represents the REST API Gateway HTTP server.
type Server struct {
	gqlClient      graphql.Client
	auditClient    *services.TemporalAuditService
	authMiddleware *middleware.AuthMiddleware
	httpServer     *http.Server
	redisClient    *redis.Client
	config         Config
}

// New creates a new Server instance.
func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	gqlClient := createGraphQLClient(cfg)
	internalAPIClient := createInternalAPIClient()
	auditService := services.NewTemporalAuditService(internalAPIClient, cfg.InternalAPIURL)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v. Using in-memory fallback for session storage.", err)
	} else {
		log.Println("Connected to Redis")
	}

	// Create JWT verifier for Oathkeeper tokens
	authMiddleware := middleware.NewAuthMiddleware()

	return &Server{
		config:         *cfg,
		gqlClient:      gqlClient,
		auditClient:    auditService,
		authMiddleware: authMiddleware,
		redisClient:    redisClient,
	}, nil
}

// Start configures the HTTP routing stack and starts serving requests.
func (s *Server) Start() error {
	// Rate limiting removed - now handled by Traefik at edge level
	// All requests are rate limited by Traefik before reaching this service

	auditService := s.auditClient
	if auditService == nil {
		log.Println("Warning: audit service not available, audit logging disabled")
	}

	userHandlers := handlers.NewUserHandlers(s.gqlClient)
	// TODO: Remove old OAuth handlers - now using Kratos
	// authHandlers := handlers.NewAuthHandlers(
	// 	s.gqlClient,
	// 	s.redisClient,
	// 	auditService,
	// 	s.config.GoogleClientID,
	// 	s.config.GoogleClientSecret,
	// 	s.config.GoogleRedirectURL,
	// 	s.config.JWTSecret,
	// 	s.config.FrontendURL,
	// )
	meHandlers := handlers.NewMeHandlers(auditService, s.config.KratosAdminURL)

	mux := http.NewServeMux()

	// TODO: Remove old OAuth routes - now using Kratos
	// mux.Handle("GET /api/auth/google/login", http.HandlerFunc(authHandlers.GoogleLogin))
	// mux.Handle("GET /api/auth/google/callback", http.HandlerFunc(authHandlers.GoogleCallback))

	// Health check endpoint - no auth required, rate limited by Traefik
	mux.Handle("GET /health", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("health check write error: %v", err)
		}
	}))

	// Authorization is now handled by Keto directly via Oathkeeper

	// User endpoints - auth required, rate limited by Traefik
	mux.Handle("GET /api/me", s.authMiddleware.RequireAuth(http.HandlerFunc(meHandlers.GetMe)))
	mux.Handle("POST /api/auth/logout", s.authMiddleware.RequireAuth(http.HandlerFunc(meHandlers.Logout)))
	mux.Handle("GET /api/auth/token", s.authMiddleware.RequireAuth(http.HandlerFunc(meHandlers.GetToken)))

	mux.Handle("GET /api/users", s.authMiddleware.RequireAuth(http.HandlerFunc(userHandlers.ListUsers)))
	mux.Handle("GET /api/users/{id}", s.authMiddleware.RequireAuth(http.HandlerFunc(userHandlers.GetUser)))
	mux.Handle("GET /api/users/email/{email}", s.authMiddleware.RequireAuth(http.HandlerFunc(userHandlers.GetUserByEmail)))

	mux.Handle("POST /api/users", s.authMiddleware.RequireAuth(http.HandlerFunc(userHandlers.CreateUser)))
	mux.Handle("PUT /api/users/{id}", s.authMiddleware.RequireAuth(http.HandlerFunc(userHandlers.UpdateUser)))
	mux.Handle("DELETE /api/users/{id}", s.authMiddleware.RequireAuth(http.HandlerFunc(userHandlers.DeleteUser)))

	var handler http.Handler = mux
	handler = middleware.Recovery(handler)
	handler = middleware.Logger(handler)
	handler = middleware.CORS(s.config.AllowedOrigins)(handler)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	fmt.Printf("REST API Gateway starting on http://%s\n", addr)
	if err := s.httpServer.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %w", err)
		}
	}
	return nil
}

// Shutdown gracefully stops the server and releases resources.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			log.Printf("Error closing Redis client: %v", err)
		}
	}

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
	}
	return nil
}

func createGraphQLClient(cfg *Config) graphql.Client {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	return gql.NewClientWithHTTPClient(cfg.GraphQLEndpoint, httpClient)
}

func createInternalAPIClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}
