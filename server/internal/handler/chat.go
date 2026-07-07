package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekhrunov/messenger/server/internal/node"
	"github.com/ekhrunov/messenger/server/internal/repository"
	"github.com/ekhrunov/messenger/server/internal/service"
	"github.com/ekhrunov/messenger/server/internal/ws"
)

type ChatHandler struct {
	service *service.ChatService
	hub     *ws.Hub
	node    node.Registry
}

func NewChatHandler(chatService *service.ChatService, hub *ws.Hub, nodeRegistry node.Registry) *ChatHandler {
	return &ChatHandler{service: chatService, hub: hub, node: nodeRegistry}
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

type chatMemberRequest struct {
	UserID                string `json:"user_id"`
	WrappedChatPrivateKey []byte `json:"wrapped_chat_private_key"`
	KemCiphertext         []byte `json:"kem_ciphertext"`
}

type createChatRequest struct {
	Name         string              `json:"name"`
	KemPublicKey []byte              `json:"kem_public_key"`
	Members      []chatMemberRequest `json:"members"`
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

	members := make([]service.ChatMemberKeyInput, 0, len(req.Members))
	userIDs := make([]string, 0, len(req.Members))
	for _, member := range req.Members {
		userID, err := h.node.LocalID(member.UserID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		userIDs = append(userIDs, userID)
		members = append(members, service.ChatMemberKeyInput{
			UserID:                userID,
			WrappedChatPrivateKey: member.WrappedChatPrivateKey,
			KemCiphertext:         member.KemCiphertext,
		})
	}

	chat, err := h.service.Create(r.Context(), service.CreateChatInput{
		Name:         req.Name,
		KemPublicKey: req.KemPublicKey,
		Members:      members,
	}, tokenUser)
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
