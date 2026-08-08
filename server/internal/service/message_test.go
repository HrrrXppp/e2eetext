package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

type messageRepoStub struct{}

func (s *messageRepoStub) List(_ context.Context, _ repository.MessageFilter) ([]domain.Message, error) {
	return nil, nil
}

func (s *messageRepoStub) Create(_ context.Context, _ domain.Message) (domain.Message, error) {
	return domain.Message{}, nil
}

func (s *messageRepoStub) GetByID(_ context.Context, id string) (domain.Message, error) {
	return domain.Message{ID: id, ChatID: "11111111-1111-1111-1111-111111111111"}, nil
}

func (s *messageRepoStub) MarkRead(_ context.Context, _, _ string) error {
	return nil
}

type messageChatRepoStub struct {
	memberships map[string]map[string]bool
}

func (s *messageChatRepoStub) List(_ context.Context, _ repository.ChatFilter) ([]domain.Chat, error) {
	return nil, nil
}

func (s *messageChatRepoStub) Create(_ context.Context, _ string, _ []string, _ int, _ string) (domain.Chat, error) {
	return domain.Chat{}, nil
}

func (s *messageChatRepoStub) UserBelongsToChat(_ context.Context, chatID, userID string) (bool, error) {
	users, ok := s.memberships[chatID]
	if !ok {
		return false, nil
	}
	return users[userID], nil
}

func (s *messageChatRepoStub) UnreadCountsForChat(_ context.Context, _ string) ([]repository.ChatUnreadCount, error) {
	return nil, nil
}

func (s *messageChatRepoStub) ListUnreadChatsForUser(_ context.Context, _ string) ([]repository.UserChatUnread, error) {
	return nil, nil
}

func (s *messageChatRepoStub) ListMembers(_ context.Context, _ string) ([]domain.ChatMember, error) {
	return nil, nil
}

type messageUserRepoStub struct {
	users []domain.User
}

func (s *messageUserRepoStub) List(_ context.Context, filter repository.UserFilter) ([]domain.User, error) {
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

func (s *messageUserRepoStub) Search(_ context.Context, _ repository.UserSearchFilter, _ string) ([]domain.User, error) {
	return nil, nil
}

func (s *messageUserRepoStub) Create(_ context.Context, user domain.User) (domain.User, error) {
	return user, nil
}

func (s *messageUserRepoStub) GetByID(_ context.Context, id string) (domain.User, error) {
	return domain.User{ID: id}, nil
}

func (s *messageUserRepoStub) UpdateName(_ context.Context, id string, name string) (domain.User, error) {
	return domain.User{ID: id, Name: name}, nil
}

type messageOIDCProviderRepoStub struct {
	providers []domain.OIDCProvider
}

func (s *messageOIDCProviderRepoStub) List(_ context.Context) ([]domain.OIDCProvider, error) {
	return s.providers, nil
}

func (s *messageOIDCProviderRepoStub) GetByName(_ context.Context, name string) (domain.OIDCProvider, error) {
	for _, provider := range s.providers {
		if strings.EqualFold(provider.Name, name) {
			return provider, nil
		}
	}
	return domain.OIDCProvider{}, errors.New("oidc provider not found")
}

func TestMessageService_List_Validation(t *testing.T) {
	svc := NewMessageService(&messageRepoStub{}, &messageChatRepoStub{}, &messageUserRepoStub{}, &messageOIDCProviderRepoStub{})

	_, err := svc.List(context.Background(), repository.MessageFilter{}, TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if err == nil || err.Error() != "chat_id is required" {
		t.Fatalf("List() error = %v", err)
	}
}

func TestMessageService_List_DeniesOtherChat(t *testing.T) {
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	svc := NewMessageService(&messageRepoStub{}, &messageChatRepoStub{
		memberships: map[string]map[string]bool{},
	}, &messageUserRepoStub{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &messageOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	_, err := svc.List(context.Background(), repository.MessageFilter{ChatID: chatID}, TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if !errors.Is(err, ErrMessageAccessDenied) {
		t.Fatalf("List() error = %v, want %v", err, ErrMessageAccessDenied)
	}
}

func TestMessageService_Create_Validation(t *testing.T) {
	svc := NewMessageService(&messageRepoStub{}, &messageChatRepoStub{}, &messageUserRepoStub{}, &messageOIDCProviderRepoStub{})

	_, err := svc.Create(context.Background(), CreateMessageInput{
		ChatID: "11111111-1111-1111-1111-111111111111",
	}, TokenUser{})
	if err == nil || err.Error() != "user_id is required" {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestMessageService_Create_DeniesOtherChat(t *testing.T) {
	const (
		chatID     = "11111111-1111-1111-1111-111111111111"
		userID     = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
		providerID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	)

	svc := NewMessageService(&messageRepoStub{}, &messageChatRepoStub{
		memberships: map[string]map[string]bool{},
	}, &messageUserRepoStub{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &messageOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	_, err := svc.Create(context.Background(), CreateMessageInput{
		ChatID: chatID,
		UserID: userID,
		Data:   "hello",
	}, TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if !errors.Is(err, ErrMessageAccessDenied) {
		t.Fatalf("Create() error = %v, want %v", err, ErrMessageAccessDenied)
	}
}

func TestMessageService_MarkRead(t *testing.T) {
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	messageID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	svc := NewMessageService(&messageRepoStub{}, &messageChatRepoStub{
		memberships: map[string]map[string]bool{
			chatID: {userID: true},
		},
	}, &messageUserRepoStub{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &messageOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	message, err := svc.MarkRead(context.Background(), messageID, TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}
	if message.Unread {
		t.Fatal("Unread = true, want false")
	}
}

type notFoundMessageRepoStub struct {
	messageRepoStub
}

func (s *notFoundMessageRepoStub) GetByID(_ context.Context, _ string) (domain.Message, error) {
	return domain.Message{}, repository.ErrMessageNotFound
}

func TestMessageService_MarkRead_NotFound(t *testing.T) {
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	svc := NewMessageService(
		&notFoundMessageRepoStub{},
		&messageChatRepoStub{
			memberships: map[string]map[string]bool{
				"11111111-1111-1111-1111-111111111111": {userID: true},
			},
		},
		&messageUserRepoStub{
			users: []domain.User{
				{
					ID:             userID,
					OIDCProviderID: providerID,
					Subject:        "google-subject-1",
				},
			},
		},
		&messageOIDCProviderRepoStub{
			providers: []domain.OIDCProvider{
				{ID: providerID, Name: "Google"},
			},
		},
	)

	_, err := svc.MarkRead(context.Background(), "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("MarkRead() error = %v, want %v", err, ErrMessageNotFound)
	}
}

func TestMessageService_MarkRead_AccessDenied(t *testing.T) {
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	messageID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	svc := NewMessageService(&messageRepoStub{}, &messageChatRepoStub{
		memberships: map[string]map[string]bool{},
	}, &messageUserRepoStub{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &messageOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	_, err := svc.MarkRead(context.Background(), messageID, TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if !errors.Is(err, ErrMessageAccessDenied) {
		t.Fatalf("MarkRead() error = %v, want %v", err, ErrMessageAccessDenied)
	}
	_ = chatID
}

type enrichingMessageRepoStub struct {
	messageRepoStub
}

func (s *enrichingMessageRepoStub) Create(_ context.Context, message domain.Message) (domain.Message, error) {
	message.ID = "33333333-3333-3333-3333-333333333333"
	return message, nil
}

type enrichingUserRepoStub struct {
	messageUserRepoStub
}

func (s *enrichingUserRepoStub) GetByID(_ context.Context, id string) (domain.User, error) {
	return domain.User{ID: id, Name: "Alice"}, nil
}

func TestMessageService_Create_EnrichesUserName(t *testing.T) {
	const (
		chatID     = "11111111-1111-1111-1111-111111111111"
		userID     = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
		providerID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	)

	svc := NewMessageService(&enrichingMessageRepoStub{}, &messageChatRepoStub{
		memberships: map[string]map[string]bool{
			chatID: {userID: true},
		},
	}, &enrichingUserRepoStub{
		messageUserRepoStub: messageUserRepoStub{
			users: []domain.User{
				{
					ID:             userID,
					OIDCProviderID: providerID,
					Subject:        "google-subject-1",
					Name:           "Alice",
				},
			},
		},
	}, &messageOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	message, err := svc.Create(context.Background(), CreateMessageInput{
		ChatID: chatID,
		UserID: userID,
		Data:   "hello",
	}, TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if message.UserName != "Alice" {
		t.Fatalf("UserName = %q, want Alice", message.UserName)
	}
}
