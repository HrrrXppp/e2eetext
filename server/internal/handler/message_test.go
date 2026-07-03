package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
	"github.com/ekhrunov/messenger/server/internal/service"
	"github.com/ekhrunov/messenger/server/internal/ws"
)

type stubMessageRepo struct {
	messagesByChatID map[string][]domain.Message
}

func (s *stubMessageRepo) List(_ context.Context, filter repository.MessageFilter) ([]domain.Message, error) {
	messages, ok := s.messagesByChatID[filter.ChatID]
	if !ok {
		return []domain.Message{}, nil
	}
	return messages, nil
}

func (s *stubMessageRepo) Create(_ context.Context, message domain.Message) (domain.Message, error) {
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	message.ID = "33333333-3333-3333-3333-333333333333"
	message.CreatedAt = createdAt
	message.UpdatedAt = createdAt
	return message, nil
}

func (s *stubMessageRepo) GetByID(_ context.Context, id string) (domain.Message, error) {
	for _, messages := range s.messagesByChatID {
		for _, message := range messages {
			if message.ID == id {
				return message, nil
			}
		}
	}
	return domain.Message{}, repository.ErrMessageNotFound
}

func (s *stubMessageRepo) MarkRead(_ context.Context, _, _ string) error {
	return nil
}

type stubMessageChatRepo struct {
	memberships map[string]map[string]bool
}

func (s *stubMessageChatRepo) List(_ context.Context, _ repository.ChatFilter) ([]domain.Chat, error) {
	return nil, nil
}

func (s *stubMessageChatRepo) Create(_ context.Context, _ string, _ []string) (domain.Chat, error) {
	return domain.Chat{}, nil
}

func (s *stubMessageChatRepo) UserBelongsToChat(_ context.Context, chatID, userID string) (bool, error) {
	users, ok := s.memberships[chatID]
	if !ok {
		return false, nil
	}
	return users[userID], nil
}

func (s *stubMessageChatRepo) UnreadCountsForChat(_ context.Context, _ string) ([]repository.ChatUnreadCount, error) {
	return nil, nil
}

func (s *stubMessageChatRepo) ListUnreadChatsForUser(_ context.Context, _ string) ([]repository.UserChatUnread, error) {
	return nil, nil
}

type stubMessageUserRepo struct {
	users []domain.User
}

func (s *stubMessageUserRepo) List(_ context.Context, filter repository.UserFilter) ([]domain.User, error) {
	var users []domain.User
	for _, user := range s.users {
		if filter.Subject != "" && user.Subject != filter.Subject {
			continue
		}
		if filter.OIDCProviderID != "" && user.OIDCProviderID != filter.OIDCProviderID {
			continue
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *stubMessageUserRepo) Search(_ context.Context, _ repository.UserSearchFilter, _ string) ([]domain.User, error) {
	return nil, nil
}

func (s *stubMessageUserRepo) Create(_ context.Context, user domain.User) (domain.User, error) {
	return user, nil
}

func (s *stubMessageUserRepo) GetByID(_ context.Context, id string) (domain.User, error) {
	return domain.User{ID: id}, nil
}

func (s *stubMessageUserRepo) UpdateName(_ context.Context, id string, name string) (domain.User, error) {
	return domain.User{ID: id, Name: name}, nil
}

type stubMessageOIDCProviderRepo struct {
	providers []domain.OIDCProvider
}

func (s *stubMessageOIDCProviderRepo) List(_ context.Context) ([]domain.OIDCProvider, error) {
	return s.providers, nil
}

func (s *stubMessageOIDCProviderRepo) GetByName(_ context.Context, name string) (domain.OIDCProvider, error) {
	for _, provider := range s.providers {
		if strings.EqualFold(provider.Name, name) {
			return provider, nil
		}
	}
	return domain.OIDCProvider{}, errors.New("oidc provider not found")
}

func newMessageHandlerForTest(
	messageRepo *stubMessageRepo,
	chatRepo *stubMessageChatRepo,
	userRepo *stubMessageUserRepo,
	oidcRepo *stubMessageOIDCProviderRepo,
) *MessageHandler {
	if chatRepo == nil {
		chatRepo = &stubMessageChatRepo{}
	}
	if userRepo == nil {
		userRepo = &stubMessageUserRepo{}
	}
	if oidcRepo == nil {
		oidcRepo = &stubMessageOIDCProviderRepo{}
	}
	return NewMessageHandler(
		service.NewMessageService(messageRepo, chatRepo, userRepo, oidcRepo),
		chatRepo,
		ws.NewHub(),
		testNode(),
	)
}

func setScopedMessagePath(req *http.Request, localMessageID string) {
	req.SetPathValue("nodeId", testNodeID)
	req.SetPathValue("localId", localMessageID)
}

func TestMessageHandler_List(t *testing.T) {
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	repo := &stubMessageRepo{
		messagesByChatID: map[string][]domain.Message{
			chatID: {
				{
					ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					ChatID:    chatID,
					UserID:    userID,
					Data:      "hello",
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
				},
			},
		},
	}
	handler := newMessageHandlerForTest(repo, &stubMessageChatRepo{
		memberships: map[string]map[string]bool{
			chatID: {userID: true},
		},
	}, &stubMessageUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &stubMessageOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodGet, "/api/v1/message?chat_id="+scopedID(chatID), nil),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var messages []domain.Message
	if err := json.NewDecoder(rec.Body).Decode(&messages); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	if messages[0].Data != "hello" {
		t.Errorf("Data = %q", messages[0].Data)
	}
	if messages[0].ID != scopedID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa") {
		t.Errorf("ID = %q", messages[0].ID)
	}
	if messages[0].ChatID != scopedID(chatID) {
		t.Errorf("ChatID = %q", messages[0].ChatID)
	}
	if messages[0].UserID != scopedID(userID) {
		t.Errorf("UserID = %q", messages[0].UserID)
	}
}

func TestMessageHandler_List_ForbiddenForOtherChat(t *testing.T) {
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	handler := newMessageHandlerForTest(&stubMessageRepo{}, &stubMessageChatRepo{
		memberships: map[string]map[string]bool{},
	}, &stubMessageUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &stubMessageOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodGet, "/api/v1/message?chat_id="+scopedID(chatID), nil),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestMessageHandler_List_MissingChatID(t *testing.T) {
	handler := newMessageHandlerForTest(&stubMessageRepo{}, nil, nil, nil)

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodGet, "/api/v1/message", nil),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMessageHandler_List_UnauthorizedWithoutTokenUser(t *testing.T) {
	handler := newMessageHandlerForTest(&stubMessageRepo{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/message?chat_id="+scopedID("11111111-1111-1111-1111-111111111111"), nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMessageHandler_Create(t *testing.T) {
	const (
		chatID     = "11111111-1111-1111-1111-111111111111"
		userID     = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
		providerID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	)

	handler := newMessageHandlerForTest(&stubMessageRepo{}, &stubMessageChatRepo{
		memberships: map[string]map[string]bool{
			chatID: {userID: true},
		},
	}, &stubMessageUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &stubMessageOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	body := []byte(`{
		"chat_id": "` + scopedID(chatID) + `",
		"user_id": "` + scopedID(userID) + `",
		"data": "hello world"
	}`)
	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodPost, "/api/v1/message", bytes.NewReader(body)),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var message domain.Message
	if err := json.NewDecoder(rec.Body).Decode(&message); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if message.Data != "hello world" {
		t.Errorf("Data = %q", message.Data)
	}
	if message.ID != scopedID("33333333-3333-3333-3333-333333333333") {
		t.Errorf("ID = %q", message.ID)
	}
	if message.ChatID != scopedID(chatID) {
		t.Errorf("ChatID = %q", message.ChatID)
	}
	if message.UserID != scopedID(userID) {
		t.Errorf("UserID = %q", message.UserID)
	}
}

func TestMessageHandler_Create_ForbiddenForOtherChat(t *testing.T) {
	const (
		chatID     = "11111111-1111-1111-1111-111111111111"
		userID     = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
		providerID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	)

	handler := newMessageHandlerForTest(&stubMessageRepo{}, &stubMessageChatRepo{
		memberships: map[string]map[string]bool{},
	}, &stubMessageUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &stubMessageOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	body := []byte(`{
		"chat_id": "` + scopedID(chatID) + `",
		"user_id": "` + scopedID(userID) + `",
		"data": "hello world"
	}`)
	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodPost, "/api/v1/message", bytes.NewReader(body)),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestMessageHandler_Create_ValidationError(t *testing.T) {
	handler := newMessageHandlerForTest(&stubMessageRepo{}, nil, nil, nil)

	req := requestWithTokenUser(
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/message",
			bytes.NewReader([]byte(`{"chat_id":"`+scopedID("11111111-1111-1111-1111-111111111111")+`"}`)),
		),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMessageHandler_Update_MarksMessageRead(t *testing.T) {
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	messageID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	repo := &stubMessageRepo{
		messagesByChatID: map[string][]domain.Message{
			chatID: {
				{
					ID:        messageID,
					ChatID:    chatID,
					UserID:    userID,
					Data:      "hello",
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
					Unread:    true,
				},
			},
		},
	}
	handler := newMessageHandlerForTest(repo, &stubMessageChatRepo{
		memberships: map[string]map[string]bool{
			chatID: {userID: true},
		},
	}, &stubMessageUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &stubMessageOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodPatch, "/api/v1/message/"+scopedID(messageID), bytes.NewReader([]byte(`{"unread":false}`))),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	setScopedMessagePath(req, messageID)
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var message domain.Message
	if err := json.NewDecoder(rec.Body).Decode(&message); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if message.Unread {
		t.Fatal("Unread = true, want false")
	}
	if message.ID != scopedID(messageID) {
		t.Errorf("ID = %q", message.ID)
	}
	if message.ChatID != scopedID(chatID) {
		t.Errorf("ChatID = %q", message.ChatID)
	}
	if message.UserID != scopedID(userID) {
		t.Errorf("UserID = %q", message.UserID)
	}
}

func TestMessageHandler_Update_NotFound(t *testing.T) {
	messageID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	handler := newMessageHandlerForTest(
		&stubMessageRepo{messagesByChatID: map[string][]domain.Message{}},
		&stubMessageChatRepo{memberships: map[string]map[string]bool{
			"11111111-1111-1111-1111-111111111111": {userID: true},
		}},
		&stubMessageUserRepo{
			users: []domain.User{
				{
					ID:             userID,
					OIDCProviderID: providerID,
					Subject:        "google-subject-1",
				},
			},
		},
		&stubMessageOIDCProviderRepo{
			providers: []domain.OIDCProvider{
				{ID: providerID, Name: "Google"},
			},
		},
	)

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodPatch, "/api/v1/message/"+scopedID(messageID), bytes.NewReader([]byte(`{"unread":false}`))),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	setScopedMessagePath(req, messageID)
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMessageHandler_Update_RejectUnreadTrue(t *testing.T) {
	messageID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	handler := newMessageHandlerForTest(&stubMessageRepo{}, nil, nil, nil)

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodPatch, "/api/v1/message/"+scopedID(messageID), bytes.NewReader([]byte(`{"unread":true}`))),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	setScopedMessagePath(req, messageID)
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMessageHandler_Update_MissingUnreadField(t *testing.T) {
	messageID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	handler := newMessageHandlerForTest(&stubMessageRepo{}, nil, nil, nil)

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodPatch, "/api/v1/message/"+scopedID(messageID), bytes.NewReader([]byte(`{}`))),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	setScopedMessagePath(req, messageID)
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
