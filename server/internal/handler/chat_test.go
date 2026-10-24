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
	admins        map[string]bool
	members       []domain.ChatMember
	removeErr     error
	removedIDs    []string
}

func (s *stubChatRepo) IsAdmin(_ context.Context, _, userID string) (bool, error) {
	return s.admins[userID], nil
}

func (s *stubChatRepo) ListMembers(_ context.Context, _ string) ([]domain.ChatMember, error) {
	return s.members, nil
}

func (s *stubChatRepo) RemoveMember(_ context.Context, _, userID string) error {
	if s.removeErr != nil {
		return s.removeErr
	}
	s.removedIDs = append(s.removedIDs, userID)
	return nil
}

func (s *stubChatRepo) List(_ context.Context, filter repository.ChatFilter) ([]domain.Chat, error) {
	chats, ok := s.chatsByUserID[filter.UserID]
	if !ok {
		return []domain.Chat{}, nil
	}
	return chats, nil
}

func (s *stubChatRepo) UserBelongsToChat(_ context.Context, chatID, userID string) (bool, error) {
	if chats, ok := s.chatsByUserID[userID]; ok {
		for _, chat := range chats {
			if chat.ID == chatID {
				return true, nil
			}
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

func (s *stubChatRepo) Create(_ context.Context, name string, userIDs []string, disappearAfterMinutes int, createdBy string) (domain.Chat, error) {
	if disappearAfterMinutes == 0 {
		disappearAfterMinutes = domain.DefaultDisappearAfterMinutes
	}
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	chat := domain.Chat{
		ID:                    "33333333-3333-3333-3333-333333333333",
		Name:                  name,
		DisappearAfterMinutes: disappearAfterMinutes,
		CreatedBy:             createdBy,
		IsAdmin:               true,
		CreatedAt:             createdAt,
		UpdatedAt:             createdAt,
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

type stubE2EERepo struct {
	chatRepo *stubChatRepo
}

func (s *stubE2EERepo) UpsertIdentityKey(context.Context, string, string, json.RawMessage) error {
	return nil
}

func (s *stubE2EERepo) GetIdentityKey(context.Context, string) (domain.UserIdentityKey, error) {
	return domain.UserIdentityKey{}, errors.New("identity key not found")
}

func (s *stubE2EERepo) CreateChatWithKeys(ctx context.Context, name string, userIDs []string, _, createdBy string, _ map[string]json.RawMessage, disappearAfterMinutes int) (domain.Chat, error) {
	return s.chatRepo.Create(ctx, name, userIDs, disappearAfterMinutes, createdBy)
}

func (s *stubE2EERepo) ListKeyWrapsForUser(context.Context, string, string) ([]domain.UserChatKeyWrap, error) {
	return nil, nil
}

func newChatHandlerForTest(chatRepo *stubChatRepo, userRepo *stubChatUserRepo, oidcRepo *stubChatOIDCProviderRepo) *ChatHandler {
	if userRepo == nil {
		userRepo = &stubChatUserRepo{}
	}
	if oidcRepo == nil {
		oidcRepo = &stubChatOIDCProviderRepo{}
	}
	e2eeRepo := &stubE2EERepo{chatRepo: chatRepo}
	e2eeService := service.NewE2EEService(e2eeRepo, chatRepo, userRepo, oidcRepo)
	return NewChatHandler(service.NewChatService(chatRepo, userRepo, oidcRepo), e2eeService, nil, testNode())
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
		],
		"e2ee": {
			"key_id": "11111111-1111-1111-1111-111111111111",
			"wraps": [
				{"user_id": "` + currentUserID + `", "wrap": {"v":1,"alg":"hybrid-kem-mlkem768-x25519-aes256gcm","keyId":"11111111-1111-1111-1111-111111111111","kemCiphertext":"a","nonce":"b","ciphertext":"c"}},
				{"user_id": "` + otherUserID + `", "wrap": {"v":1,"alg":"hybrid-kem-mlkem768-x25519-aes256gcm","keyId":"11111111-1111-1111-1111-111111111111","kemCiphertext":"d","nonce":"e","ciphertext":"f"}}
			]
		}
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
		],
		"e2ee": {
			"key_id": "11111111-1111-1111-1111-111111111111",
			"wraps": [
				{"user_id": "` + otherUserID + `", "wrap": {"v":1,"alg":"hybrid-kem-mlkem768-x25519-aes256gcm","keyId":"11111111-1111-1111-1111-111111111111","kemCiphertext":"a","nonce":"b","ciphertext":"c"}},
				{"user_id": "` + thirdUserID + `", "wrap": {"v":1,"alg":"hybrid-kem-mlkem768-x25519-aes256gcm","keyId":"11111111-1111-1111-1111-111111111111","kemCiphertext":"d","nonce":"e","ciphertext":"f"}}
			]
		}
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

func membersTestHandler(repo *stubChatRepo) *ChatHandler {
	const (
		callerID   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		providerID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	)
	return newChatHandlerForTest(repo, &stubChatUserRepo{
		users: []domain.User{{ID: callerID, OIDCProviderID: providerID, Subject: "google-subject-1"}},
	}, &stubChatOIDCProviderRepo{providers: []domain.OIDCProvider{{ID: providerID, Name: "Google"}}})
}

func membersTestRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.SetPathValue("nodeId", testNodeID)
	req.SetPathValue("localId", "11111111-1111-1111-1111-111111111111")
	req.SetPathValue("userNodeId", testNodeID)
	req.SetPathValue("userLocalId", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	return requestWithTokenUser(req, service.TokenUser{Subject: "google-subject-1", Provider: "google"})
}

func memberChats(chatID, userID string) map[string][]domain.Chat {
	return map[string][]domain.Chat{userID: {{ID: chatID}}}
}

func TestChatHandler_ListMembers(t *testing.T) {
	const chatID = "11111111-1111-1111-1111-111111111111"
	const callerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	repo := &stubChatRepo{
		chatsByUserID: memberChats(chatID, callerID),
		members: []domain.ChatMember{
			{User: domain.User{ID: callerID, Subject: "s1", Name: "Alice"}, IsAdmin: true},
			{User: domain.User{ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Subject: "s2", Name: "Bob"}},
		},
	}
	rec := httptest.NewRecorder()
	membersTestHandler(repo).ListMembers(rec, membersTestRequest(http.MethodGet, "/api/v1/chat/x/y/members"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		IsAdmin bool   `json:"isAdmin"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 || got[0].ID != scopedID(callerID) || !got[0].IsAdmin || got[1].IsAdmin || got[1].Name != "Bob" {
		t.Fatalf("members = %+v", got)
	}
}

func TestChatHandler_ListMembers_ForbiddenForNonMember(t *testing.T) {
	rec := httptest.NewRecorder()
	membersTestHandler(&stubChatRepo{}).ListMembers(rec, membersTestRequest(http.MethodGet, "/api/v1/chat/x/y/members"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestChatHandler_RemoveMember(t *testing.T) {
	const chatID = "11111111-1111-1111-1111-111111111111"
	const callerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	const targetID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	tests := []struct {
		name       string
		repo       *stubChatRepo
		path       func(*http.Request)
		wantStatus int
		wantRemove int
	}{
		{
			name:       "admin removes member",
			repo:       &stubChatRepo{chatsByUserID: memberChats(chatID, callerID), admins: map[string]bool{callerID: true}},
			wantStatus: http.StatusNoContent,
			wantRemove: 1,
		},
		{
			name:       "non-admin forbidden",
			repo:       &stubChatRepo{chatsByUserID: memberChats(chatID, callerID)},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "non-member forbidden",
			repo:       &stubChatRepo{admins: map[string]bool{callerID: true}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "self removal rejected",
			repo:       &stubChatRepo{chatsByUserID: memberChats(chatID, callerID), admins: map[string]bool{callerID: true}},
			path:       func(r *http.Request) { r.SetPathValue("userLocalId", callerID) },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "target not a member",
			repo:       &stubChatRepo{chatsByUserID: memberChats(chatID, callerID), admins: map[string]bool{callerID: true}, removeErr: repository.ErrChatMemberNotFound},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "last admin protected",
			repo:       &stubChatRepo{chatsByUserID: memberChats(chatID, callerID), admins: map[string]bool{callerID: true}, removeErr: repository.ErrLastChatAdmin},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "repository failure",
			repo:       &stubChatRepo{chatsByUserID: memberChats(chatID, callerID), admins: map[string]bool{callerID: true}, removeErr: errors.New("boom")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := membersTestRequest(http.MethodDelete, "/api/v1/chat/x/y/members/a/b")
			if tt.path != nil {
				tt.path(req)
			}
			rec := httptest.NewRecorder()
			membersTestHandler(tt.repo).RemoveMember(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if len(tt.repo.removedIDs) != tt.wantRemove {
				t.Fatalf("removed = %v, want %d call(s)", tt.repo.removedIDs, tt.wantRemove)
			}
		})
	}
}
