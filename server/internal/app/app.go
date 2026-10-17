package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/handler"
	"github.com/ekhrunov/messenger/server/internal/metrics"
	"github.com/ekhrunov/messenger/server/internal/middleware"
	"github.com/ekhrunov/messenger/server/internal/node"
	"github.com/ekhrunov/messenger/server/internal/repository/postgres"
	"github.com/ekhrunov/messenger/server/internal/service"
	"github.com/ekhrunov/messenger/server/internal/ws"
)

type App struct {
	handler http.Handler
	db      *sql.DB

	// metricsPublisher is nil unless CloudWatch metrics are configured.
	metricsPublisher *metrics.Publisher
}

func New(cfg config.Config, logger *slog.Logger, db *sql.DB, nodeRegistry node.Registry) (*App, error) {
	oidcProviderRepo := postgres.NewOIDCProviderRepository(db)
	userRepo := postgres.NewUserRepository(db)
	chatRepo := postgres.NewChatRepository(db)
	messageRepo := postgres.NewMessageRepository(db)
	e2eeRepo := postgres.NewE2EERepository(db)
	authService := service.NewAuthService(cfg, oidcProviderRepo)
	userService := service.NewUserService(userRepo, oidcProviderRepo)
	chatService := service.NewChatService(chatRepo, userRepo, oidcProviderRepo)
	e2eeService := service.NewE2EEService(e2eeRepo, chatRepo, userRepo, oidcProviderRepo)
	messageService := service.NewMessageService(messageRepo, chatRepo, userRepo, oidcProviderRepo)
	hub := ws.NewHub()
	requireAuth := middleware.RequireAuth(authService)

	// Rate limits by route class (see config.RateLimits for the defaults).
	// Auth and WS run before any user identity exists, so they key by IP;
	// Search, Message, and Default run after RequireAuth and key by user,
	// falling back to IP for the handful of authenticated handlers that
	// don't fit a more specific class.
	authLimit := middleware.RateLimit(toMiddlewareRateLimit(cfg.RateLimits.Auth), middleware.IPKey)
	searchLimit := middleware.RateLimit(toMiddlewareRateLimit(cfg.RateLimits.Search), middleware.UserKey)
	messageLimit := middleware.RateLimit(toMiddlewareRateLimit(cfg.RateLimits.Message), middleware.UserKey)
	defaultLimit := middleware.RateLimit(toMiddlewareRateLimit(cfg.RateLimits.Default), middleware.UserKey)
	wsLimit := middleware.RateLimit(toMiddlewareRateLimit(cfg.RateLimits.WS), middleware.IPKey)

	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.NewHealthHandler())

	authHandler := handler.NewAuthHandler(authService, nodeRegistry)
	mux.Handle("GET /api/v1/auth/providers", authLimit(http.HandlerFunc(authHandler.ListProviders)))
	mux.Handle("GET /api/v1/auth/{provider}/login", authLimit(http.HandlerFunc(authHandler.Login)))
	mux.Handle("GET /api/v1/auth/{provider}/callback", authLimit(http.HandlerFunc(authHandler.Callback)))
	// POST callback: required by providers using response_mode=form_post
	// (Apple, whenever name/email scopes are requested) — the provider POSTs
	// code/state/the one-time "user" field to this same URL instead of a
	// 302 redirect with query params.
	mux.Handle("POST /api/v1/auth/{provider}/callback", authLimit(http.HandlerFunc(authHandler.Callback)))
	mux.Handle("POST /api/v1/auth/refresh", authLimit(http.HandlerFunc(authHandler.Refresh)))

	userHandler := handler.NewUserHandler(userService, nodeRegistry)
	mux.Handle("GET /api/v1/search", requireAuth(searchLimit(http.HandlerFunc(userHandler.Search))))
	mux.Handle("GET /api/v1/user", requireAuth(defaultLimit(http.HandlerFunc(userHandler.List))))
	mux.Handle("POST /api/v1/user", requireAuth(defaultLimit(http.HandlerFunc(userHandler.Create))))
	mux.Handle("GET /api/v1/user/{nodeId}/{localId}", requireAuth(defaultLimit(http.HandlerFunc(userHandler.GetByID))))
	mux.Handle("PATCH /api/v1/user/{nodeId}/{localId}", requireAuth(defaultLimit(http.HandlerFunc(userHandler.Update))))

	chatHandler := handler.NewChatHandler(chatService, e2eeService, hub, nodeRegistry)
	mux.Handle("GET /api/v1/chat", requireAuth(defaultLimit(http.HandlerFunc(chatHandler.List))))
	mux.Handle("POST /api/v1/chat", requireAuth(defaultLimit(http.HandlerFunc(chatHandler.Create))))

	e2eeHandler := handler.NewE2EEHandler(e2eeService, nodeRegistry)
	mux.Handle("PUT /api/v1/user/{nodeId}/{localId}/identity-key", requireAuth(defaultLimit(http.HandlerFunc(e2eeHandler.PutIdentityKey))))
	mux.Handle("GET /api/v1/user/{nodeId}/{localId}/identity-key", requireAuth(defaultLimit(http.HandlerFunc(e2eeHandler.GetIdentityKey))))
	mux.Handle("GET /api/v1/chat/{nodeId}/{localId}/key-wraps", requireAuth(defaultLimit(http.HandlerFunc(e2eeHandler.ListKeyWraps))))

	messageHandler := handler.NewMessageHandler(messageService, chatRepo, hub, nodeRegistry)
	mux.Handle("GET /api/v1/message", requireAuth(messageLimit(http.HandlerFunc(messageHandler.List))))
	mux.Handle("POST /api/v1/message", requireAuth(messageLimit(http.HandlerFunc(messageHandler.Create))))
	mux.Handle("PATCH /api/v1/message/{nodeId}/{localId}", requireAuth(messageLimit(http.HandlerFunc(messageHandler.Update))))

	wsHandler := handler.NewWSHandler(authService, logger, hub, userRepo, oidcProviderRepo)
	mux.Handle("GET /api/v1/ws", wsLimit(http.HandlerFunc(wsHandler.ServeHTTP)))

	root := middleware.Logging(logger)(mux)

	application := &App{handler: root, db: db}

	if cfg.Metrics.Enabled() {
		awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(cfg.Metrics.Region))
		if err != nil {
			return nil, fmt.Errorf("load aws config for metrics: %w", err)
		}
		logger.Info("cloudwatch metrics enabled", "namespace", cfg.Metrics.Namespace, "environment", cfg.Metrics.Environment)
		application.metricsPublisher = metrics.NewPublisher(
			cloudwatch.NewFromConfig(awsCfg),
			cfg.Metrics.Namespace,
			cfg.Metrics.Environment,
			nodeRegistry.ID,
			hub.ConnectionCount,
			metrics.DefaultInterval,
			logger,
		)
	}

	return application, nil
}

// StartBackground starts the app's background workers (currently the optional
// CloudWatch metrics publisher). They stop when ctx is cancelled.
func (a *App) StartBackground(ctx context.Context) {
	if a.metricsPublisher != nil {
		go a.metricsPublisher.Run(ctx)
	}
}

func (a *App) DB() *sql.DB {
	return a.db
}

func (a *App) Handler() http.Handler {
	return a.handler
}

// toMiddlewareRateLimit adapts config.RateLimitConfig to
// middleware.RateLimitConfig. The two packages each define their own copy of
// this shape (config can't import middleware: middleware depends on service,
// which already depends on config) rather than share a type across layers.
func toMiddlewareRateLimit(cfg config.RateLimitConfig) middleware.RateLimitConfig {
	return middleware.RateLimitConfig{RequestsPerMinute: cfg.RequestsPerMinute, Burst: cfg.Burst}
}
