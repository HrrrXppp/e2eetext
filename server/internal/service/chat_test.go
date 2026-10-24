package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

type chatRepoStub struct {
	members    map[string]bool
	admins     map[string]bool
	removeErr  error
	removedIDs []string
}

func (s *chatRepoStub) List(_ context.Context, _ repository.ChatFilter) ([]domain.Chat, error) {
	return nil, nil
}

func (s *chatRepoStub) Create(_ context.Context, _ string, _ []string, _ int, _ string) (domain.Chat, error) {
	return domain.Chat{}, nil
}

func (s *chatRepoStub) UserBelongsToChat(_ context.Context, _, userID string) (bool, error) {
	return s.members[userID], nil
}

func (s *chatRepoStub) IsAdmin(_ context.Context, _, userID string) (bool, error) {
	return s.admins[userID], nil
}

func (s *chatRepoStub) ListMembers(_ context.Context, _ string) ([]domain.ChatMember, error) {
	return []domain.ChatMember{{User: domain.User{ID: "m1"}, IsAdmin: true}}, nil
}

func (s *chatRepoStub) RemoveMember(_ context.Context, _, userID string) error {
	if s.removeErr != nil {
		return s.removeErr
	}
	s.removedIDs = append(s.removedIDs, userID)
	return nil
}

func (s *chatRepoStub) UnreadCountsForChat(_ context.Context, _ string) ([]repository.ChatUnreadCount, error) {
	return nil, nil
}

func (s *chatRepoStub) ListUnreadChatsForUser(_ context.Context, _ string) ([]repository.UserChatUnread, error) {
	return nil, nil
}

type chatUserRepoStub struct {
	users []domain.User
}

func (s *chatUserRepoStub) List(_ context.Context, filter repository.UserFilter) ([]domain.User, error) {
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

func (s *chatUserRepoStub) Search(_ context.Context, _ repository.UserSearchFilter, _ string) ([]domain.User, error) {
	return nil, nil
}

func (s *chatUserRepoStub) Create(_ context.Context, user domain.User) (domain.User, error) {
	return user, nil
}

func (s *chatUserRepoStub) GetByID(_ context.Context, id string) (domain.User, error) {
	return domain.User{ID: id}, nil
}

func (s *chatUserRepoStub) UpdateName(_ context.Context, id string, name string) (domain.User, error) {
	return domain.User{ID: id, Name: name}, nil
}

type chatOIDCProviderRepoStub struct {
	providers []domain.OIDCProvider
}

func (s *chatOIDCProviderRepoStub) List(_ context.Context) ([]domain.OIDCProvider, error) {
	return s.providers, nil
}

func (s *chatOIDCProviderRepoStub) GetByName(_ context.Context, name string) (domain.OIDCProvider, error) {
	for _, provider := range s.providers {
		if strings.EqualFold(provider.Name, name) {
			return provider, nil
		}
	}
	return domain.OIDCProvider{}, errors.New("oidc provider not found")
}

func TestChatService_Create_Validation(t *testing.T) {
	svc := NewChatService(&chatRepoStub{}, &chatUserRepoStub{}, &chatOIDCProviderRepoStub{})

	_, err := svc.Create(context.Background(), CreateChatInput{
		UsersUIDs: []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
	}, TokenUser{})
	if err == nil || err.Error() != "name is required" {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.Create(context.Background(), CreateChatInput{
		Name: "Project Chat",
	}, TokenUser{})
	if err == nil || err.Error() != "users_uids is required" {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestChatService_Create_DeniesWhenCurrentUserNotInMembers(t *testing.T) {
	const (
		currentUserID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		otherUserID   = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
		providerID    = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	)

	svc := NewChatService(&chatRepoStub{}, &chatUserRepoStub{
		users: []domain.User{
			{
				ID:             currentUserID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &chatOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	_, err := svc.Create(context.Background(), CreateChatInput{
		Name:      "Project Chat",
		UsersUIDs: []string{otherUserID},
	}, TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if !errors.Is(err, ErrChatAccessDenied) {
		t.Fatalf("Create() error = %v, want %v", err, ErrChatAccessDenied)
	}
}

func TestChatService_List_Validation(t *testing.T) {
	svc := NewChatService(&chatRepoStub{}, &chatUserRepoStub{}, &chatOIDCProviderRepoStub{})

	_, err := svc.List(context.Background(), repository.ChatFilter{}, TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if err == nil || err.Error() != "user_id is required" {
		t.Fatalf("List() error = %v", err)
	}
}

func TestChatService_List_DeniesOtherUser(t *testing.T) {
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	providerID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	svc := NewChatService(&chatRepoStub{}, &chatUserRepoStub{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}, &chatOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "Google"},
		},
	})

	_, err := svc.List(context.Background(), repository.ChatFilter{
		UserID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	}, TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if !errors.Is(err, ErrChatAccessDenied) {
		t.Fatalf("List() error = %v, want %v", err, ErrChatAccessDenied)
	}
}

const (
	rmChatID    = "11111111-1111-1111-1111-111111111111"
	rmAdminID   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	rmMemberID  = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	rmOutsider  = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	rmProvider  = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	rmSubjAdmin = "sub-admin"
)

func newRemoveMemberService(repo *chatRepoStub, callerID, subject string) (*ChatService, TokenUser) {
	svc := NewChatService(repo, &chatUserRepoStub{users: []domain.User{
		{ID: callerID, OIDCProviderID: rmProvider, Subject: subject},
	}}, &chatOIDCProviderRepoStub{providers: []domain.OIDCProvider{{ID: rmProvider, Name: "Google"}}})
	return svc, TokenUser{Subject: subject, Provider: "google"}
}

func TestChatService_RemoveMember(t *testing.T) {
	tests := []struct {
		name      string
		repo      *chatRepoStub
		callerID  string
		targetID  string
		wantErr   error
		wantCalls int
	}{
		{
			name:      "admin removes member",
			repo:      &chatRepoStub{members: map[string]bool{rmAdminID: true, rmMemberID: true}, admins: map[string]bool{rmAdminID: true}},
			callerID:  rmAdminID,
			targetID:  rmMemberID,
			wantCalls: 1,
		},
		{
			name:     "non-admin member denied",
			repo:     &chatRepoStub{members: map[string]bool{rmAdminID: true, rmMemberID: true}, admins: map[string]bool{}},
			callerID: rmAdminID,
			targetID: rmMemberID,
			wantErr:  ErrChatNotAdmin,
		},
		{
			name:     "non-member caller denied",
			repo:     &chatRepoStub{members: map[string]bool{rmMemberID: true}, admins: map[string]bool{rmOutsider: true}},
			callerID: rmOutsider,
			targetID: rmMemberID,
			wantErr:  ErrChatAccessDenied,
		},
		{
			name:     "self removal rejected",
			repo:     &chatRepoStub{members: map[string]bool{rmAdminID: true}, admins: map[string]bool{rmAdminID: true}},
			callerID: rmAdminID,
			targetID: rmAdminID,
			wantErr:  ErrChatSelfRemoval,
		},
		{
			name:     "target not a member",
			repo:     &chatRepoStub{members: map[string]bool{rmAdminID: true}, admins: map[string]bool{rmAdminID: true}, removeErr: repository.ErrChatMemberNotFound},
			callerID: rmAdminID,
			targetID: rmOutsider,
			wantErr:  repository.ErrChatMemberNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, tokenUser := newRemoveMemberService(tt.repo, tt.callerID, rmSubjAdmin)
			err := svc.RemoveMember(context.Background(), rmChatID, tt.targetID, tokenUser)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RemoveMember() error = %v, want %v", err, tt.wantErr)
			}
			if len(tt.repo.removedIDs) != tt.wantCalls {
				t.Fatalf("RemoveMember repo calls = %d, want %d", len(tt.repo.removedIDs), tt.wantCalls)
			}
		})
	}
}

func TestChatService_ListMembers(t *testing.T) {
	repo := &chatRepoStub{members: map[string]bool{rmMemberID: true}}

	svc, tokenUser := newRemoveMemberService(repo, rmMemberID, rmSubjAdmin)
	members, err := svc.ListMembers(context.Background(), rmChatID, tokenUser)
	if err != nil || len(members) != 1 {
		t.Fatalf("ListMembers() = %v, %v", members, err)
	}

	svc, tokenUser = newRemoveMemberService(repo, rmOutsider, rmSubjAdmin)
	if _, err := svc.ListMembers(context.Background(), rmChatID, tokenUser); !errors.Is(err, ErrChatAccessDenied) {
		t.Fatalf("ListMembers() non-member error = %v, want %v", err, ErrChatAccessDenied)
	}
}
