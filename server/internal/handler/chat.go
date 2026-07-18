package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/node"
	"github.com/ekhrunov/messenger/server/internal/repository"
	"github.com/ekhrunov/messenger/server/internal/service"
	"github.com/ekhrunov/messenger/server/internal/ws"
)

type ChatHandler struct {
	service     *service.ChatService
	e2eeService *service.E2EEService
	hub         *ws.Hub
	node        node.Registry
}

func NewChatHandler(
	chatService *service.ChatService,
	e2eeService *service.E2EEService,
	hub *ws.Hub,
	nodeRegistry node.Registry,
) *ChatHandler {
	return &ChatHandler{service: chatService, e2eeService: e2eeService, hub: hub, node: nodeRegistry}
}

func (h *ChatHandler) List(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	userID, err := h.node.LocalID(strings.TrimSpace(r.URL.Query().Get("user_id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	filter := repository.ChatFilter{
		UserID: userID,
	}

	chats, err := h.service.List(r.Context(), filter, tokenUser)
	if err != nil {
		if errors.Is(err, service.ErrChatAccessDenied) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if strings.Contains(err.Error(), "is required") {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIChats(h.node, chats))
}

type createChatRequest struct {
	UsersUIDs             []string        `json:"users_uids"`
	Name                  string          `json:"name"`
	DisappearAfterMinutes *int            `json:"disappear_after_minutes,omitempty"`
	E2EE                  *createChatE2EE `json:"e2ee,omitempty"`
}

type createChatE2EE struct {
	KeyID string              `json:"key_id"`
	Wraps []createChatKeyWrap `json:"wraps"`
}

type createChatKeyWrap struct {
	UserID string          `json:"user_id"`
	Wrap   json.RawMessage `json:"wrap"`
}

func (h *ChatHandler) Create(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var req createChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	userIDs, err := localIDs(h.node, req.UsersUIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.E2EE == nil {
		writeError(w, http.StatusBadRequest, errors.New("e2ee is required"))
		return
	}
	if h.e2eeService == nil {
		writeError(w, http.StatusBadRequest, errors.New("e2ee is not configured"))
		return
	}

	wraps := make([]service.ChatKeyWrapInput, 0, len(req.E2EE.Wraps))
	for _, item := range req.E2EE.Wraps {
		userID, wrapErr := h.node.LocalID(strings.TrimSpace(item.UserID))
		if wrapErr != nil {
			writeError(w, http.StatusBadRequest, wrapErr)
			return
		}
		wraps = append(wraps, service.ChatKeyWrapInput{
			UserID: userID,
			Wrap:   item.Wrap,
		})
	}

	keyID := strings.TrimSpace(req.E2EE.KeyID)
	if keyID == "" {
		writeError(w, http.StatusBadRequest, errors.New("e2ee.key_id is required"))
		return
	}

	disappearAfterMinutes := domain.DefaultDisappearAfterMinutes
	if req.DisappearAfterMinutes != nil {
		disappearAfterMinutes = *req.DisappearAfterMinutes
	}
	if disappearAfterMinutes < 1 {
		writeError(w, http.StatusBadRequest, errors.New("disappear_after_minutes must be positive"))
		return
	}

	chat, err := h.e2eeService.CreateChat(r.Context(), service.CreateE2EEChatInput{
		Name:                  req.Name,
		UsersUIDs:             userIDs,
		KeyID:                 keyID,
		Wraps:                 wraps,
		DisappearAfterMinutes: disappearAfterMinutes,
	}, tokenUser)
	if err != nil {
		if errors.Is(err, service.ErrChatAccessDenied) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if isE2EEChatValidationError(err) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIChat(h.node, chat))
	h.notifyChatAdded(userIDs, chat.ID)
}

func (h *ChatHandler) notifyChatAdded(userIDs []string, chatID string) {
	if h.hub == nil {
		return
	}

	event := ws.ChatAddedEvent{
		Type:   ws.EventTypeChatAdded,
		ChatID: h.node.ScopeID(chatID),
	}

	for _, userID := range userIDs {
		h.hub.NotifyChatAdded(userID, event)
	}
}

func isE2EEChatValidationError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "is required") ||
		strings.Contains(msg, "e2ee.wrap") ||
		strings.Contains(msg, "e2ee.key_id") ||
		strings.Contains(msg, "too large") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "missing e2ee")
}
