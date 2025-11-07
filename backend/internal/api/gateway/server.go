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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/retran/nexus/backend/internal/api/common/middleware"
	"github.com/retran/nexus/backend/internal/api/gateway/handlers"
	"github.com/retran/nexus/backend/internal/audit"
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
	auditClient    audit.Service
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
	auditService := audit.NewClient(internalAPIClient, cfg.InternalAPIURL)

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
	auditService := s.auditClient
	if auditService == nil {
		log.Println("Warning: audit service not available, audit logging disabled")
	}

	meHandlers := handlers.NewMeHandlers(auditService)

	mux := http.NewServeMux()

	mux.Handle("GET /health", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("health check write error: %v", err)
		}
	}))

	mux.Handle("GET /api/me", s.authMiddleware.RequireAuth(http.HandlerFunc(meHandlers.GetMe)))

	var handler http.Handler = mux
	handler = middleware.Recovery(handler)
	handler = middleware.Logger(handler)
	handler = middleware.CORS(s.config.AllowedOrigins)(handler)
	// Wrap with OpenTelemetry middleware for distributed tracing
	handler = otelhttp.NewHandler(handler, "nexus-gateway")

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
