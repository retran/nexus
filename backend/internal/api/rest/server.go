// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

package rest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/client"

	"github.com/retran/nexus/backend/internal/api/rest/handlers"
	"github.com/retran/nexus/backend/internal/api/rest/middleware"
	"github.com/retran/nexus/backend/internal/api/rest/services"
	gql "github.com/retran/nexus/backend/internal/client/graphql"
	"github.com/retran/nexus/backend/internal/repository/postgres"
)

// Config contains REST API Gateway server configuration.
type Config struct {
	TemporalNamespace  string
	GoogleClientSecret string
	TemporalTaskQueue  string
	TemporalHost       string
	GoogleClientID     string
	GraphQLEndpoint    string
	JWTSecret          string
	DatabaseURL        string
	RedisHost          string
	GoogleRedirectURL  string
	Host               string
	RedisPassword      string
	FrontendURL        string
	AllowedOrigins     []string
	RedisPort          int
	RedisDB            int
	ShutdownTimeout    time.Duration
	WriteTimeout       time.Duration
	Port               int
	ReadTimeout        time.Duration
	RateLimitOAuth     int
	RateLimitHealth    int
	RateLimitAPI       int
	RateLimitAdmin     int
}

// Server represents the REST API Gateway HTTP server.
type Server struct {
	gqlClient graphql.Client
	pool      interface {
		postgres.DBTX
		Close()
	}
	temporalClient client.Client
	httpServer     *http.Server
	redisClient    *redis.Client
	db             *postgres.Queries
	config         Config
}

func New(cfg Config) (*Server, error) {
	gqlClient := gql.NewClient(cfg.GraphQLEndpoint)

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

	var temporalClient client.Client
	if cfg.TemporalHost != "" {
		log.Printf("Connecting to Temporal at %s...", cfg.TemporalHost)
		temporalClient, err = client.Dial(client.Options{
			HostPort:  cfg.TemporalHost,
			Namespace: cfg.TemporalNamespace,
		})
		if err != nil {
			log.Printf("Warning: Failed to connect to Temporal: %v. Audit logging will be disabled.", err)
			temporalClient = nil
		} else {
			log.Println("Connected to Temporal")
		}
	}

	return &Server{
		config:         cfg,
		gqlClient:      gqlClient,
		redisClient:    redisClient,
		db:             db,
		pool:           pool,
		temporalClient: temporalClient,
	}, nil
}

func (s *Server) Start() error {
	authMiddleware := middleware.NewAuthMiddleware(s.gqlClient, s.config.JWTSecret)

	// TODO: Remove after OAuth migration to Kratos is complete
	_ = middleware.NewRateLimiter(s.redisClient, middleware.RateLimitConfig{
		Requests: s.config.RateLimitOAuth,
		Window:   time.Minute,
		KeyFunc:  middleware.DefaultKeyFunc,
	})

	healthRateLimiter := middleware.NewRateLimiter(s.redisClient, middleware.RateLimitConfig{
		Requests: s.config.RateLimitHealth,
		Window:   time.Minute,
		KeyFunc:  middleware.DefaultKeyFunc,
	})

	apiRateLimiter := middleware.NewRateLimiter(s.redisClient, middleware.RateLimitConfig{
		Requests: s.config.RateLimitAPI,
		Window:   time.Minute,
		KeyFunc:  middleware.UserKeyFunc,
	})

	adminRateLimiter := middleware.NewRateLimiter(s.redisClient, middleware.RateLimitConfig{
		Requests: s.config.RateLimitAdmin,
		Window:   time.Minute,
		KeyFunc:  middleware.UserKeyFunc,
	})

	var auditService *services.TemporalAuditService
	if s.temporalClient != nil {
		auditService = services.NewTemporalAuditService(s.temporalClient, s.config.TemporalTaskQueue)
	} else {
		log.Println("Warning: Temporal client not available, audit logging disabled")
		auditService = nil
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
	kratosWebhookHandlers := handlers.NewKratosWebhookHandlers(s.gqlClient)

	mux := http.NewServeMux()

	// Kratos webhook (no auth required - validated by webhook secret)
	mux.Handle("POST /api/webhooks/kratos/registration", apiRateLimiter.Middleware(http.HandlerFunc(kratosWebhookHandlers.HandleRegistration)))

	// TODO: Remove old OAuth routes - now using Kratos
	// mux.Handle("GET /api/auth/google/login", oauthRateLimiter.Middleware(http.HandlerFunc(authHandlers.GoogleLogin)))
	// mux.Handle("GET /api/auth/google/callback", oauthRateLimiter.Middleware(http.HandlerFunc(authHandlers.GoogleCallback)))

	mux.Handle("GET /health", healthRateLimiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})))

	mux.Handle("GET /api/me", apiRateLimiter.Middleware(authMiddleware.RequireAuth(http.HandlerFunc(meHandlers.GetMe))))
	mux.Handle("POST /api/auth/logout", apiRateLimiter.Middleware(authMiddleware.RequireAuth(http.HandlerFunc(meHandlers.Logout))))
	mux.Handle("GET /api/auth/token", apiRateLimiter.Middleware(http.HandlerFunc(meHandlers.GetToken)))

	mux.Handle("GET /api/users", apiRateLimiter.Middleware(authMiddleware.RequireAuth(http.HandlerFunc(userHandlers.ListUsers))))
	mux.Handle("GET /api/users/{id}", apiRateLimiter.Middleware(authMiddleware.RequireAuth(http.HandlerFunc(userHandlers.GetUser))))
	mux.Handle("GET /api/users/email/{email}", apiRateLimiter.Middleware(authMiddleware.RequireAuth(http.HandlerFunc(userHandlers.GetUserByEmail))))

	mux.Handle("POST /api/users", adminRateLimiter.Middleware(authMiddleware.RequireAdmin(http.HandlerFunc(userHandlers.CreateUser))))
	mux.Handle("PUT /api/users/{id}", adminRateLimiter.Middleware(authMiddleware.RequireAdmin(http.HandlerFunc(userHandlers.UpdateUser))))
	mux.Handle("DELETE /api/users/{id}", adminRateLimiter.Middleware(authMiddleware.RequireAdmin(http.HandlerFunc(userHandlers.DeleteUser))))

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
	return s.httpServer.ListenAndServe()
}

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
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
