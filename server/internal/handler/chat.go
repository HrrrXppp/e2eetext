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

func (h *ChatHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	chatID, err := h.node.LocalID(scopedIDFromPath(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	members, err := h.service.ListMembers(r.Context(), chatID, tokenUser)
	if err != nil {
		if errors.Is(err, service.ErrChatAccessDenied) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if errors.Is(err, service.ErrChatIDRequired) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIChatMembers(h.node, members))
}

type addMembersRequest struct {
	UserIDs []string            `json:"user_ids"`
	KeyID   string              `json:"key_id"`
	Wraps   []createChatKeyWrap `json:"wraps"`
}

// AddMembers adds one or more users to an existing chat together with a new chat key version wrapped for every
// current and new member. Only chat admins may call it.
func (h *ChatHandler) AddMembers(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	if h.e2eeService == nil {
		writeError(w, http.StatusBadRequest, errors.New("e2ee is not configured"))
		return
	}

	chatID, err := h.node.LocalID(scopedIDFromPath(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req addMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	userIDs, err := localIDs(h.node, req.UserIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	wraps := make([]service.ChatKeyWrapInput, 0, len(req.Wraps))
	for _, item := range req.Wraps {
		userID, wrapErr := h.node.LocalID(strings.TrimSpace(item.UserID))
		if wrapErr != nil {
			writeError(w, http.StatusBadRequest, wrapErr)
			return
		}
		wraps = append(wraps, service.ChatKeyWrapInput{UserID: userID, Wrap: item.Wrap})
	}

	err = h.e2eeService.AddMembers(r.Context(), chatID, userIDs, req.KeyID, wraps, tokenUser)
	if err != nil {
		var already *service.AlreadyMembersError
		var missing *service.MissingIdentityKeysError
		switch {
		case errors.Is(err, service.ErrE2EEAccessDenied), errors.Is(err, service.ErrChatNotAdmin):
			writeError(w, http.StatusForbidden, err)
		case errors.As(err, &already):
			h.writeUserListError(w, http.StatusConflict, err, already.UserIDs)
		case errors.Is(err, service.ErrAlreadyMember):
			writeError(w, http.StatusConflict, err)
		case errors.Is(err, service.ErrKeyVersionExists):
			writeError(w, http.StatusConflict, err)
		case errors.As(err, &missing):
			h.writeUserListError(w, http.StatusBadRequest, err, missing.UserIDs)
		case isAddMembersValidationError(err):
			writeError(w, http.StatusBadRequest, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
	h.notifyChatAdded(userIDs, chatID)
}

// writeUserListError replies with the error text followed by the offending scoped user ids, also returned as user_ids.
func (h *ChatHandler) writeUserListError(w http.ResponseWriter, status int, err error, localUserIDs []string) {
	scoped := make([]string, len(localUserIDs))
	for i, id := range localUserIDs {
		scoped[i] = h.node.ScopeID(id)
	}
	writeJSON(w, status, map[string]any{
		"error":    err.Error() + ": " + strings.Join(scoped, ", "),
		"user_ids": scoped,
	})
}

func isAddMembersValidationError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "is required") ||
		strings.Contains(msg, "wraps must include") ||
		strings.Contains(msg, "not a chat member") ||
		strings.Contains(msg, "too large") ||
		strings.Contains(msg, "too many") ||
		strings.Contains(msg, "duplicate")
}

func (h *ChatHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	chatID, err := h.node.LocalID(scopedIDFromPath(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	userID, err := h.node.LocalID(scopedIDFromPathValues(r, "userNodeId", "userLocalId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.service.RemoveMember(r.Context(), chatID, userID, tokenUser); err != nil {
		switch {
		case errors.Is(err, service.ErrChatAccessDenied), errors.Is(err, service.ErrChatNotAdmin):
			writeError(w, http.StatusForbidden, err)
		case errors.Is(err, repository.ErrChatMemberNotFound):
			// Same status and message as any other denial, so callers cannot probe which accounts belong to a chat.
			writeError(w, http.StatusForbidden, service.ErrChatAccessDenied)
		case errors.Is(err, service.ErrChatSelfRemoval), errors.Is(err, repository.ErrLastChatAdmin),
			errors.Is(err, service.ErrChatIDRequired), errors.Is(err, service.ErrChatUserIDRequired):
			writeError(w, http.StatusBadRequest, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
	if h.hub != nil {
		h.hub.NotifyChatRemoved(userID, ws.ChatRemovedEvent{
			Type:   ws.EventTypeChatRemoved,
			ChatID: h.node.ScopeID(chatID),
		})
	}
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
