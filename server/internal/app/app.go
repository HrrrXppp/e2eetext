package app

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/handler"
	"github.com/ekhrunov/messenger/server/internal/middleware"
	"github.com/ekhrunov/messenger/server/internal/node"
	"github.com/ekhrunov/messenger/server/internal/repository/postgres"
	"github.com/ekhrunov/messenger/server/internal/service"
	"github.com/ekhrunov/messenger/server/internal/ws"
)

type App struct {
	handler http.Handler
	db      *sql.DB
}

func New(cfg config.Config, logger *slog.Logger, db *sql.DB, nodeRegistry node.Registry) (*App, error) {
	oidcProviderRepo := postgres.NewOIDCProviderRepository(db)
	userRepo := postgres.NewUserRepository(db)
	chatRepo := postgres.NewChatRepository(db)
	messageRepo := postgres.NewMessageRepository(db)
	authService := service.NewAuthService(cfg, oidcProviderRepo)
	userService := service.NewUserService(userRepo, oidcProviderRepo)
	chatService := service.NewChatService(chatRepo, userRepo, oidcProviderRepo)
	messageService := service.NewMessageService(messageRepo, chatRepo, userRepo, oidcProviderRepo)
	hub := ws.NewHub()
	requireAuth := middleware.RequireAuth(authService)

	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.NewHealthHandler())

	authHandler := handler.NewAuthHandler(authService, nodeRegistry)
	mux.HandleFunc("GET /api/v1/auth/providers", authHandler.ListProviders)
	mux.HandleFunc("GET /api/v1/auth/{provider}/login", authHandler.Login)
	mux.HandleFunc("GET /api/v1/auth/{provider}/callback", authHandler.Callback)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)

	userHandler := handler.NewUserHandler(userService, nodeRegistry)
	mux.Handle("GET /api/v1/search", requireAuth(http.HandlerFunc(userHandler.Search)))
	mux.Handle("GET /api/v1/user", requireAuth(http.HandlerFunc(userHandler.List)))
	mux.Handle("POST /api/v1/user", requireAuth(http.HandlerFunc(userHandler.Create)))
	mux.Handle("GET /api/v1/user/{nodeId}/{localId}", requireAuth(http.HandlerFunc(userHandler.GetByID)))
	mux.Handle("PATCH /api/v1/user/{nodeId}/{localId}", requireAuth(http.HandlerFunc(userHandler.Update)))

	chatHandler := handler.NewChatHandler(chatService, hub, nodeRegistry)
	mux.Handle("GET /api/v1/chat", requireAuth(http.HandlerFunc(chatHandler.List)))
	mux.Handle("POST /api/v1/chat", requireAuth(http.HandlerFunc(chatHandler.Create)))

	messageHandler := handler.NewMessageHandler(messageService, chatRepo, hub, nodeRegistry)
	mux.Handle("GET /api/v1/message", requireAuth(http.HandlerFunc(messageHandler.List)))
	mux.Handle("POST /api/v1/message", requireAuth(http.HandlerFunc(messageHandler.Create)))
	mux.Handle("PATCH /api/v1/message/{nodeId}/{localId}", requireAuth(http.HandlerFunc(messageHandler.Update)))

	wsHandler := handler.NewWSHandler(authService, logger, hub, userRepo, oidcProviderRepo)
	mux.HandleFunc("GET /api/v1/ws", wsHandler.ServeHTTP)

	root := middleware.Logging(logger)(mux)

	return &App{handler: root, db: db}, nil
}

func (a *App) DB() *sql.DB {
	return a.db
}

func (a *App) Handler() http.Handler {
	return a.handler
}
