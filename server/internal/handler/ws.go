package handler

import (
	"log/slog"
	"net/http"

	"github.com/ekhrunov/messenger/server/internal/repository"
	"github.com/ekhrunov/messenger/server/internal/requestorigin"
	"github.com/ekhrunov/messenger/server/internal/service"
	"github.com/ekhrunov/messenger/server/internal/ws"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	auth             *service.AuthService
	logger           *slog.Logger
	hub              *ws.Hub
	userRepo         repository.UserRepository
	oidcProviderRepo repository.OIDCProviderRepository
	upgrader         websocket.Upgrader
}

func NewWSHandler(
	auth *service.AuthService,
	logger *slog.Logger,
	hub *ws.Hub,
	userRepo repository.UserRepository,
	oidcProviderRepo repository.OIDCProviderRepository,
) *WSHandler {
	return &WSHandler{
		auth:             auth,
		logger:           logger,
		hub:              hub,
		userRepo:         userRepo,
		oidcProviderRepo: oidcProviderRepo,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return requestorigin.MatchesPublicOrigin(r, r.Header.Get("Origin"))
			},
		},
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tokenUser, err := h.auth.AuthenticateWebSocket(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	userID, err := service.ResolveCurrentUserID(r.Context(), tokenUser, h.userRepo, h.oidcProviderRepo)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("websocket upgrade failed", "err", err)
		return
	}

	go h.serveConn(conn, tokenUser, userID)
}

func (h *WSHandler) serveConn(conn *websocket.Conn, user service.TokenUser, userID string) {
	defer conn.Close()

	h.hub.Register(userID, conn)
	defer h.hub.Unregister(userID, conn)

	h.logger.Info("websocket connected", "user_id", userID, "subject", user.Subject, "provider", user.Provider)

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if !ws.IsHeartbeat(payload) {
			continue
		}

		ack, err := ws.HeartbeatAckPayload()
		if err != nil {
			h.logger.Warn("encode heartbeat ack failed", "err", err)
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, ack); err != nil {
			break
		}
	}

	h.logger.Info("websocket disconnected", "user_id", userID, "subject", user.Subject, "provider", user.Provider)
}
