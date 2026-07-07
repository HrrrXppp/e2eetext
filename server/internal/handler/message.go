package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekhrunov/messenger/server/internal/node"
	"github.com/ekhrunov/messenger/server/internal/repository"
	"github.com/ekhrunov/messenger/server/internal/service"
	"github.com/ekhrunov/messenger/server/internal/ws"
)

type MessageHandler struct {
	service  *service.MessageService
	chatRepo repository.ChatRepository
	hub      *ws.Hub
	node     node.Registry
}

func NewMessageHandler(
	messageService *service.MessageService,
	chatRepo repository.ChatRepository,
	hub *ws.Hub,
	nodeRegistry node.Registry,
) *MessageHandler {
	return &MessageHandler{
		service:  messageService,
		chatRepo: chatRepo,
		hub:      hub,
		node:     nodeRegistry,
	}
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	chatID, err := h.node.LocalID(strings.TrimSpace(r.URL.Query().Get("chat_id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	filter := repository.MessageFilter{
		ChatID: chatID,
	}

	messages, err := h.service.List(r.Context(), filter, tokenUser)
	if err != nil {
		if errors.Is(err, service.ErrMessageAccessDenied) || errors.Is(err, service.ErrChatAccessDenied) {
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

	writeJSON(w, http.StatusOK, toAPIMessages(h.node, messages))
}

type createMessageRequest struct {
	ChatID        string `json:"chat_id"`
	UserID        string `json:"user_id"`
	Data          string `json:"data"`
	KemCiphertext []byte `json:"kem_ciphertext"`
}

func (h *MessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var req createMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	chatID, err := h.node.LocalID(strings.TrimSpace(req.ChatID))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	userID, err := h.node.LocalID(strings.TrimSpace(req.UserID))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	message, err := h.service.Create(r.Context(), service.CreateMessageInput{
		ChatID:        chatID,
		UserID:        userID,
		Data:          strings.TrimSpace(req.Data),
		KemCiphertext: req.KemCiphertext,
	}, tokenUser)
	if err != nil {
		if errors.Is(err, service.ErrMessageAccessDenied) || errors.Is(err, service.ErrChatAccessDenied) {
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

	writeJSON(w, http.StatusCreated, toAPIMessage(h.node, message))
	h.notifyChatUnread(r.Context(), chatID)
}

type updateMessageRequest struct {
	Unread *bool `json:"unread"`
}

func (h *MessageHandler) Update(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	messageID, err := h.node.LocalID(scopedIDFromPath(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req updateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Unread == nil {
		writeError(w, http.StatusBadRequest, errors.New("unread is required"))
		return
	}
	if *req.Unread {
		writeError(w, http.StatusBadRequest, errors.New("only unread=false is supported"))
		return
	}

	message, err := h.service.MarkRead(r.Context(), messageID, tokenUser)
	if err != nil {
		if errors.Is(err, service.ErrMessageAccessDenied) || errors.Is(err, service.ErrChatAccessDenied) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if errors.Is(err, service.ErrMessageNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if strings.Contains(err.Error(), "is required") {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIMessage(h.node, message))
	h.notifyChatUnread(r.Context(), message.ChatID)
}

func (h *MessageHandler) notifyChatUnread(ctx context.Context, chatID string) {
	if h.hub == nil {
		return
	}

	members, err := h.chatRepo.UnreadCountsForChat(ctx, chatID)
	if err != nil {
		return
	}

	for _, member := range members {
		unreadChats, err := h.chatRepo.ListUnreadChatsForUser(ctx, member.UserID)
		if err != nil {
			continue
		}

		items := make([]ws.ChatUnreadItem, len(unreadChats))
		for i, chat := range unreadChats {
			items[i] = ws.ChatUnreadItem{
				ChatID:             h.node.ScopeID(chat.ChatID),
				UnreadMessageCount: chat.Count,
				UpdatedAt:          chat.UpdatedAt,
			}
		}

		h.hub.NotifyChatUnread(member.UserID, ws.ChatUnreadEvent{
			Type:  ws.EventTypeChatUnread,
			Chats: items,
		})
	}
}
