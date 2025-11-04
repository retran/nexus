// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package rest exposes configuration and startup helpers for the REST gateway.
package rest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/redis/go-redis/v9"

	"github.com/retran/nexus/backend/internal/api/rest/handlers"
	"github.com/retran/nexus/backend/internal/api/rest/middleware"
	"github.com/retran/nexus/backend/internal/api/rest/services"
	gql "github.com/retran/nexus/backend/internal/client/graphql"
	"github.com/retran/nexus/backend/internal/repository/postgres"
)

// Config contains REST API Gateway server configuration.
type Config struct {
	VaultSecretID         string
	VaultRoleID           string
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
	KratosAdminURL        string
	OathkeeperJWTIssuer   string
	OathkeeperJWTAudience string
	OathkeeperJWKSFile    string
	InternalAPIAudience   []string
	VaultAuthMountPath    string
	VaultTransitMountPath string
	VaultKVMountPath      string
	VaultAddress          string
	ServiceJWTAudience    []string
	AllowedOrigins        []string
	AdminRoles            []string
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
	gqlClient      graphql.Client
	auditClient    *services.TemporalAuditService
	authMiddleware *middleware.AuthMiddleware
	pool           interface {
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

	gqlClient, err := createGraphQLClient(cfg)
	if err != nil {
		return nil, err
	}

	internalAPIClient, err := createInternalAPIClient()
	if err != nil {
		return nil, err
	}
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

	// Create JWT verifier for Oathkeeper tokens
	// Create auth middleware (mTLS-based, no JWT verification needed)
	authMiddleware := middleware.NewAuthMiddleware()

	return &Server{
		config:         *cfg,
		gqlClient:      gqlClient,
		auditClient:    auditService,
		authMiddleware: authMiddleware,
		redisClient:    redisClient,
		db:             db,
		pool:           pool,
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
	authorizerHandler := handlers.NewAuthorizerHandler(s.config.AdminRoles)

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

	// Internal RBAC authorizer for Oathkeeper - requires auth
	mux.Handle("POST /api/internal/authorize", s.authMiddleware.RequireAuth(http.HandlerFunc(authorizerHandler.Authorize)))

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
	// Load TLS configuration for mTLS
	tlsConfig, err := loadMTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to load mTLS config: %w", err)
	}

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		TLSConfig:    tlsConfig,
	}

	fmt.Printf("REST API Gateway starting with mTLS on https://%s\n", addr)
	if err := s.httpServer.ListenAndServeTLS("/secrets/tls.crt", "/secrets/tls.key"); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve TLS: %w", err)
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

func createGraphQLClient(cfg *Config) (graphql.Client, error) {
	tlsConfig, err := loadMTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load mTLS config: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return gql.NewClientWithHTTPClient(cfg.GraphQLEndpoint, httpClient), nil
}

func createInternalAPIClient() (*http.Client, error) {
	tlsConfig, err := loadMTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load mTLS config: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}, nil
}

// loadMTLSConfig loads TLS configuration for mutual TLS authentication.
func loadMTLSConfig() (*tls.Config, error) {
	// Load CA certificate for validating client certificates
	caCert, err := os.ReadFile("/secrets/vault-ca.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
		MinVersion: tls.VersionTLS13,
	}, nil
}
