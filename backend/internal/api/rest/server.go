// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package rest exposes configuration and startup helpers for the REST gateway.
package rest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/redis/go-redis/v9"

	"github.com/retran/nexus/backend/internal/api/rest/handlers"
	"github.com/retran/nexus/backend/internal/api/rest/middleware"
	"github.com/retran/nexus/backend/internal/api/rest/services"
	"github.com/retran/nexus/backend/internal/auth"
	gql "github.com/retran/nexus/backend/internal/client/graphql"
	"github.com/retran/nexus/backend/internal/repository/postgres"
	"github.com/retran/nexus/backend/internal/secrets"
)

// Config contains REST API Gateway server configuration.
type Config struct {
	VaultSecretID         string
	VaultRoleID           string
	TemporalTaskQueue     string
	TemporalHost          string
	GoogleClientID        string
	GraphQLEndpoint       string
	DatabaseURL           string
	RedisHost             string
	GoogleRedirectURL     string
	Host                  string
	RedisPassword         string
	FrontendURL           string
	ServiceJWTIssuer      string
	ServiceJWTSubject     string
	GoogleClientSecret    string
	VaultSigningKey       string
	InternalAPIURL        string
	InternalAPIAudience   []string
	VaultAuthMountPath    string
	VaultTransitMountPath string
	VaultKVMountPath      string
	VaultAddress          string
	TemporalNamespace     string
	ServiceJWTAudience    []string
	AllowedOrigins        []string
	ShutdownTimeout       time.Duration
	WriteTimeout          time.Duration
	ReadTimeout           time.Duration
	Port                  int
	RedisDB               int
	RedisPort             int
	ServiceJWTTTL         time.Duration
}

// Server represents the REST API Gateway HTTP server.
type Server struct {
	gqlClient   graphql.Client
	auditClient *services.TemporalAuditService
	pool        interface {
		postgres.DBTX
		Close()
	}
	httpServer  *http.Server
	redisClient *redis.Client
	db          *postgres.Queries
	config      Config
}

// New creates a new Server instance.
func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	tokenClient, err := newTokenClient(cfg)
	if err != nil {
		return nil, err
	}

	gqlClient, err := createGraphQLClient(cfg, tokenClient)
	if err != nil {
		return nil, err
	}

	internalAPIClient := createInternalAPIClient(cfg, tokenClient)
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

	pool, err := postgres.NewPoolFromURL(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	db := postgres.New(pool)

	return &Server{
		config:      *cfg,
		gqlClient:   gqlClient,
		auditClient: auditService,
		redisClient: redisClient,
		db:          db,
		pool:        pool,
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
	meHandlers := handlers.NewMeHandlers(auditService)

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

	// User endpoints - auth required, rate limited by Traefik
	mux.Handle("GET /api/me", http.HandlerFunc(meHandlers.GetMe))
	mux.Handle("POST /api/auth/logout", http.HandlerFunc(meHandlers.Logout))
	mux.Handle("GET /api/auth/token", http.HandlerFunc(meHandlers.GetToken))

	mux.Handle("GET /api/users", http.HandlerFunc(userHandlers.ListUsers))
	mux.Handle("GET /api/users/{id}", http.HandlerFunc(userHandlers.GetUser))
	mux.Handle("GET /api/users/email/{email}", http.HandlerFunc(userHandlers.GetUserByEmail))

	mux.Handle("POST /api/users", http.HandlerFunc(userHandlers.CreateUser))
	mux.Handle("PUT /api/users/{id}", http.HandlerFunc(userHandlers.UpdateUser))
	mux.Handle("DELETE /api/users/{id}", http.HandlerFunc(userHandlers.DeleteUser))

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

	if s.pool != nil {
		s.pool.Close()
	}

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
	}
	return nil
}

func newTokenClient(cfg *Config) (*auth.TokenClient, error) {
	if strings.TrimSpace(cfg.VaultAddress) == "" {
		return nil, errors.New("vault address is required")
	}
	if strings.TrimSpace(cfg.VaultRoleID) == "" {
		return nil, errors.New("vault role id is required")
	}
	if strings.TrimSpace(cfg.VaultSecretID) == "" {
		return nil, errors.New("vault secret id is required")
	}
	if strings.TrimSpace(cfg.VaultSigningKey) == "" {
		return nil, errors.New("vault signing key is required")
	}

	secretsClient, err := secrets.NewClient(&secrets.Config{
		Address:       cfg.VaultAddress,
		RoleID:        cfg.VaultRoleID,
		SecretID:      cfg.VaultSecretID,
		AuthMountPath: cfg.VaultAuthMountPath,
		KVMountPath:   cfg.VaultKVMountPath,
	})
	if err != nil {
		return nil, fmt.Errorf("create vault secrets client: %w", err)
	}

	tokenClient, err := auth.NewTokenClient(&auth.TokenClientConfig{
		SecretsClient:    secretsClient,
		SigningKeyName:   cfg.VaultSigningKey,
		TransitMountPath: cfg.VaultTransitMountPath,
		Issuer:           cfg.ServiceJWTIssuer,
		VersionCacheTTL:  time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("create service token client: %w", err)
	}

	return tokenClient, nil
}

func createGraphQLClient(cfg *Config, tokenClient *auth.TokenClient) (graphql.Client, error) {
	subject := strings.TrimSpace(cfg.ServiceJWTSubject)
	if subject == "" {
		return nil, errors.New("service jwt subject is required")
	}

	audience := normalizeAudience(cfg.ServiceJWTAudience)
	if len(audience) == 0 {
		return nil, errors.New("service jwt audience is required")
	}

	tokenTransport := newServiceTokenTransport(nil, tokenClient, subject, audience, cfg.ServiceJWTTTL)
	httpClient := &http.Client{
		Transport: tokenTransport,
		Timeout:   10 * time.Second,
	}

	return gql.NewClientWithHTTPClient(cfg.GraphQLEndpoint, httpClient), nil
}

func createInternalAPIClient(cfg *Config, tokenClient *auth.TokenClient) *http.Client {
	audience := normalizeAudience(cfg.InternalAPIAudience)
	if len(audience) == 0 {
		audience = []string{"internal-api"}
	}

	subject := strings.TrimSpace(cfg.ServiceJWTSubject)
	transport := newServiceTokenTransport(nil, tokenClient, subject, audience, cfg.ServiceJWTTTL)
	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}

func normalizeAudience(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
