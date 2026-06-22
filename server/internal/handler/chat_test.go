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
)

type stubChatRepo struct {
	chatsByUserID map[string][]domain.Chat
}

func (s *stubChatRepo) List(_ context.Context, filter repository.ChatFilter) ([]domain.Chat, error) {
	chats, ok := s.chatsByUserID[filter.UserID]
	if !ok {
		return []domain.Chat{}, nil
	}
	return chats, nil
}

func (s *stubChatRepo) UserBelongsToChat(_ context.Context, chatID, userID string) (bool, error) {
	chats, ok := s.chatsByUserID[userID]
	if !ok {
		return false, nil
	}
	for _, chat := range chats {
		if chat.ID == chatID {
			return true, nil
		}
	}
	return false, nil
}

func (s *stubChatRepo) UnreadCountsForChat(_ context.Context, _ string) ([]repository.ChatUnreadCount, error) {
	return nil, nil
}

func (s *stubChatRepo) ListUnreadChatsForUser(_ context.Context, _ string) ([]repository.UserChatUnread, error) {
	return nil, nil
}

func (s *stubChatRepo) Create(_ context.Context, name string, userIDs []string) (domain.Chat, error) {
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	chat := domain.Chat{
		ID:        "33333333-3333-3333-3333-333333333333",
		Name:      name,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	for _, userID := range userIDs {
		s.chatsByUserID[userID] = append(s.chatsByUserID[userID], chat)
	}
	return chat, nil
}

type stubChatUserRepo struct {
	users []domain.User
}

func (s *stubChatUserRepo) List(_ context.Context, filter repository.UserFilter) ([]domain.User, error) {
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

func (s *stubChatUserRepo) Search(_ context.Context, _ repository.UserSearchFilter, _ string) ([]domain.User, error) {
	return nil, nil
}

func (s *stubChatUserRepo) Create(_ context.Context, user domain.User) (domain.User, error) {
	return user, nil
}

func (s *stubChatUserRepo) GetByID(_ context.Context, id string) (domain.User, error) {
	return domain.User{ID: id}, nil
}

func (s *stubChatUserRepo) UpdateName(_ context.Context, id string, name string) (domain.User, error) {
	return domain.User{ID: id, Name: name}, nil
}

type stubChatOIDCProviderRepo struct {
	providers []domain.OIDCProvider
}

func (s *stubChatOIDCProviderRepo) List(_ context.Context) ([]domain.OIDCProvider, error) {
	return s.providers, nil
}

func (s *stubChatOIDCProviderRepo) GetByName(_ context.Context, name string) (domain.OIDCProvider, error) {
	for _, provider := range s.providers {
		if strings.EqualFold(provider.Name, name) {
			return provider, nil
		}
	}
	return domain.OIDCProvider{}, errors.New("oidc provider not found")
}

func newChatHandlerForTest(chatRepo *stubChatRepo, userRepo *stubChatUserRepo, oidcRepo *stubChatOIDCProviderRepo) *ChatHandler {
	if userRepo == nil {
		userRepo = &stubChatUserRepo{}
	}
	if oidcRepo == nil {
		oidcRepo = &stubChatOIDCProviderRepo{}
	}
	return NewChatHandler(service.NewChatService(chatRepo, userRepo, oidcRepo), nil, testNode())
}

func requestWithTokenUser(req *http.Request, tokenUser service.TokenUser) *http.Request {
	ctx := context.WithValue(req.Context(), service.TokenUserContextKey, tokenUser)
	return req.WithContext(ctx)
}

func TestChatHandler_List(t *testing.T) {
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	repo := &stubChatRepo{
		chatsByUserID: map[string][]domain.Chat{
			userID: {
				{
					ID:                 "11111111-1111-1111-1111-111111111111",
					Name:               "General",
					CreatedAt:          createdAt,
					UpdatedAt:          createdAt,
					UnreadMessageCount: 2,
				},
				{
					ID:                 "22222222-2222-2222-2222-222222222222",
					Name:               "Random",
					CreatedAt:          createdAt,
					UpdatedAt:          createdAt,
					UnreadMessageCount: 0,
				},
			},
		},
	}
	handler := newChatHandlerForTest(repo, &stubChatUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &stubChatOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodGet, "/api/v1/chat?user_id="+userID, nil),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var chats []domain.Chat
	if err := json.NewDecoder(rec.Body).Decode(&chats); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("chats len = %d, want 2", len(chats))
	}
	if chats[0].ID != scopedID("11111111-1111-1111-1111-111111111111") {
		t.Errorf("first chat ID = %q", chats[0].ID)
	}
	if chats[0].Name != "General" {
		t.Errorf("first chat Name = %q", chats[0].Name)
	}
	if chats[0].UnreadMessageCount != 2 {
		t.Errorf("first chat UnreadMessageCount = %d, want 2", chats[0].UnreadMessageCount)
	}
}

func TestChatHandler_List_ForbiddenForOtherUser(t *testing.T) {
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	otherUserID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	handler := newChatHandlerForTest(&stubChatRepo{chatsByUserID: map[string][]domain.Chat{}}, &stubChatUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &stubChatOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodGet, "/api/v1/chat?user_id="+otherUserID, nil),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestChatHandler_Create(t *testing.T) {
	const (
		currentUserID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		otherUserID   = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
		providerID    = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	)

	handler := newChatHandlerForTest(&stubChatRepo{chatsByUserID: map[string][]domain.Chat{}}, &stubChatUserRepo{
		users: []domain.User{
			{
				ID:             currentUserID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &stubChatOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	body := []byte(`{
		"name": "Project Chat",
		"users_uids": [
			"` + currentUserID + `",
			"` + otherUserID + `"
		]
	}`)
	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body)),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var chat domain.Chat
	if err := json.NewDecoder(rec.Body).Decode(&chat); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if chat.Name != "Project Chat" {
		t.Errorf("Name = %q", chat.Name)
	}
	if chat.ID != scopedID("33333333-3333-3333-3333-333333333333") {
		t.Errorf("ID = %q", chat.ID)
	}
}

func TestChatHandler_Create_ForbiddenWhenCurrentUserNotInMembers(t *testing.T) {
	const (
		currentUserID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		otherUserID   = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
		thirdUserID   = "cccccccc-cccc-cccc-cccc-cccccccccccc"
		providerID    = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	)

	handler := newChatHandlerForTest(&stubChatRepo{chatsByUserID: map[string][]domain.Chat{}}, &stubChatUserRepo{
		users: []domain.User{
			{
				ID:             currentUserID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &stubChatOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	body := []byte(`{
		"name": "Project Chat",
		"users_uids": [
			"` + otherUserID + `",
			"` + thirdUserID + `"
		]
	}`)
	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body)),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestChatHandler_Create_ValidationError(t *testing.T) {
	handler := newChatHandlerForTest(&stubChatRepo{}, nil, nil)

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader([]byte(`{"name":"Only Name"}`))),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestChatHandler_List_MissingUserID(t *testing.T) {
	handler := newChatHandlerForTest(&stubChatRepo{}, nil, nil)

	req := requestWithTokenUser(
		httptest.NewRequest(http.MethodGet, "/api/v1/chat", nil),
		service.TokenUser{Subject: "google-subject-1", Provider: "google"},
	)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestChatHandler_List_UnauthorizedWithoutTokenUser(t *testing.T) {
	handler := newChatHandlerForTest(&stubChatRepo{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat?user_id=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
