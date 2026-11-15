package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

type addKeyVersionRepoStub struct {
	repository.E2EERepository
	err       error
	keyID     string
	createdBy string
	wraps     map[string]json.RawMessage
}

func (s *addKeyVersionRepoStub) AddKeyVersion(_ context.Context, _, keyID, createdBy string, wraps map[string]json.RawMessage) error {
	if s.err != nil {
		return s.err
	}
	s.keyID, s.createdBy, s.wraps = keyID, createdBy, wraps
	return nil
}

func newAddKeyVersionService(isMember bool, repo *addKeyVersionRepoStub) (*E2EEService, TokenUser) {
	const providerID = "provider-1"
	users := &chatUserRepoStub{users: []domain.User{{ID: "m1", OIDCProviderID: providerID, Subject: "subject-1"}}}
	providers := &chatOIDCProviderRepoStub{providers: []domain.OIDCProvider{{ID: providerID, Name: "Google"}}}
	chats := &chatRepoStub{members: map[string]bool{"m1": isMember}}
	return NewE2EEService(repo, chats, users, providers), TokenUser{Subject: "subject-1", Provider: "google"}
}

func TestE2EEService_AddKeyVersion(t *testing.T) {
	wrap := json.RawMessage(`{"v":1}`)
	ctx := context.Background()

	t.Run("stores the new version for the current member", func(t *testing.T) {
		repo := &addKeyVersionRepoStub{}
		svc, token := newAddKeyVersionService(true, repo)
		err := svc.AddKeyVersion(ctx, "chat-1", " key-2 ", []ChatKeyWrapInput{{UserID: "m1", Wrap: wrap}}, token)
		if err != nil {
			t.Fatalf("AddKeyVersion() error = %v", err)
		}
		if repo.keyID != "key-2" || repo.createdBy != "m1" || len(repo.wraps) != 1 {
			t.Fatalf("stored keyID=%q createdBy=%q wraps=%d", repo.keyID, repo.createdBy, len(repo.wraps))
		}
	})

	t.Run("non-member is denied", func(t *testing.T) {
		svc, token := newAddKeyVersionService(false, &addKeyVersionRepoStub{})
		err := svc.AddKeyVersion(ctx, "chat-1", "key-2", []ChatKeyWrapInput{{UserID: "m1", Wrap: wrap}}, token)
		if !errors.Is(err, ErrE2EEAccessDenied) {
			t.Fatalf("error = %v, want %v", err, ErrE2EEAccessDenied)
		}
	})

	rejected := map[string]struct {
		keyID string
		wraps []ChatKeyWrapInput
	}{
		"missing key id":     {"", []ChatKeyWrapInput{{UserID: "m1", Wrap: wrap}}},
		"no wraps":           {"key-2", nil},
		"extra member":       {"key-2", []ChatKeyWrapInput{{UserID: "m1", Wrap: wrap}, {UserID: "m2", Wrap: wrap}}},
		"wrap for stranger":  {"key-2", []ChatKeyWrapInput{{UserID: "m2", Wrap: wrap}}},
		"empty wrap":         {"key-2", []ChatKeyWrapInput{{UserID: "m1"}}},
		"blank wrap user id": {"key-2", []ChatKeyWrapInput{{UserID: " ", Wrap: wrap}}},
	}
	for name, tc := range rejected {
		t.Run(name, func(t *testing.T) {
			repo := &addKeyVersionRepoStub{}
			svc, token := newAddKeyVersionService(true, repo)
			if err := svc.AddKeyVersion(ctx, "chat-1", tc.keyID, tc.wraps, token); err == nil {
				t.Fatal("AddKeyVersion() error = nil, want validation error")
			}
			if repo.wraps != nil {
				t.Fatal("repository was called for an invalid request")
			}
		})
	}

	t.Run("duplicate key id is reported", func(t *testing.T) {
		svc, token := newAddKeyVersionService(true, &addKeyVersionRepoStub{err: repository.ErrKeyVersionExists})
		err := svc.AddKeyVersion(ctx, "chat-1", "key-2", []ChatKeyWrapInput{{UserID: "m1", Wrap: wrap}}, token)
		if !errors.Is(err, ErrKeyVersionExists) {
			t.Fatalf("error = %v, want %v", err, ErrKeyVersionExists)
		}
	})
}

type addMembersRepoStub struct {
	repository.E2EERepository
	identityKeys map[string]bool
	err          error
	called       bool
	userIDs      []string
	keyID        string
	wraps        map[string]json.RawMessage
}

func (s *addMembersRepoStub) GetIdentityKey(_ context.Context, userID string) (domain.UserIdentityKey, error) {
	if !s.identityKeys[userID] {
		return domain.UserIdentityKey{}, errors.New("identity key not found")
	}
	return domain.UserIdentityKey{UserID: userID}, nil
}

func (s *addMembersRepoStub) AddMembersWithKeyVersion(_ context.Context, _ string, userIDs []string, keyID, _ string, wraps map[string]json.RawMessage) error {
	s.called = true
	if s.err != nil {
		return s.err
	}
	s.userIDs, s.keyID, s.wraps = userIDs, keyID, wraps
	return nil
}

// addMembersChatRepo has current members m1 (the admin when isAdmin) and m2.
type addMembersChatRepo struct {
	chatRepoStub
}

func (s *addMembersChatRepo) ListMembers(_ context.Context, _ string) ([]domain.ChatMember, error) {
	return []domain.ChatMember{{User: domain.User{ID: "m1"}, IsAdmin: true}, {User: domain.User{ID: "m2"}}}, nil
}

func newAddMembersService(isMember, isAdmin bool, repo *addMembersRepoStub) (*E2EEService, TokenUser) {
	const providerID = "provider-1"
	users := &chatUserRepoStub{users: []domain.User{{ID: "m1", OIDCProviderID: providerID, Subject: "subject-1"}}}
	providers := &chatOIDCProviderRepoStub{providers: []domain.OIDCProvider{{ID: providerID, Name: "Google"}}}
	chats := &addMembersChatRepo{chatRepoStub{members: map[string]bool{"m1": isMember}, admins: map[string]bool{"m1": isAdmin}}}
	return NewE2EEService(repo, chats, users, providers), TokenUser{Subject: "subject-1", Provider: "google"}
}

func TestE2EEService_AddMembers(t *testing.T) {
	wrap := json.RawMessage(`{"v":1}`)
	ctx := context.Background()
	wrapsFor := func(ids ...string) []ChatKeyWrapInput {
		out := make([]ChatKeyWrapInput, len(ids))
		for i, id := range ids {
			out[i] = ChatKeyWrapInput{UserID: id, Wrap: wrap}
		}
		return out
	}
	newRepo := func() *addMembersRepoStub {
		return &addMembersRepoStub{identityKeys: map[string]bool{"n1": true, "n2": true, "n3": true}}
	}

	t.Run("admin adds several users with one key version", func(t *testing.T) {
		repo := newRepo()
		svc, token := newAddMembersService(true, true, repo)
		err := svc.AddMembers(ctx, "chat-1", []string{"n1", " n2 "}, " key-2 ", wrapsFor("m1", "m2", "n1", "n2"), token)
		if err != nil {
			t.Fatalf("AddMembers() error = %v", err)
		}
		if repo.keyID != "key-2" || !slices.Equal(repo.userIDs, []string{"n1", "n2"}) || len(repo.wraps) != 4 {
			t.Fatalf("stored keyID=%q userIDs=%v wraps=%d", repo.keyID, repo.userIDs, len(repo.wraps))
		}
	})

	t.Run("non-member is denied", func(t *testing.T) {
		repo := newRepo()
		svc, token := newAddMembersService(false, true, repo)
		err := svc.AddMembers(ctx, "chat-1", []string{"n1"}, "key-2", wrapsFor("m1", "m2", "n1"), token)
		if !errors.Is(err, ErrE2EEAccessDenied) || repo.called {
			t.Fatalf("error = %v, called = %v", err, repo.called)
		}
	})

	t.Run("non-admin is denied", func(t *testing.T) {
		repo := newRepo()
		svc, token := newAddMembersService(true, false, repo)
		err := svc.AddMembers(ctx, "chat-1", []string{"n1"}, "key-2", wrapsFor("m1", "m2", "n1"), token)
		if !errors.Is(err, ErrChatNotAdmin) || repo.called {
			t.Fatalf("error = %v, called = %v", err, repo.called)
		}
	})

	tooMany := make([]string, 101)
	for i := range tooMany {
		tooMany[i] = "u" + strings.Repeat("x", i+1)
	}
	rejected := map[string]struct {
		keyID   string
		userIDs []string
		wraps   []ChatKeyWrapInput
	}{
		"missing key id":     {"", []string{"n1"}, wrapsFor("m1", "m2", "n1")},
		"empty user list":    {"key-2", nil, wrapsFor("m1", "m2")},
		"blank user id":      {"key-2", []string{" "}, wrapsFor("m1", "m2")},
		"duplicate user ids": {"key-2", []string{"n1", "n1"}, wrapsFor("m1", "m2", "n1")},
		"batch too large":    {"key-2", tooMany, nil},
		"missing wrap":       {"key-2", []string{"n1", "n2"}, wrapsFor("m1", "m2", "n1")},
		"extra wrap":         {"key-2", []string{"n1"}, wrapsFor("m1", "m2", "n1", "n3")},
		"duplicate wrap":     {"key-2", []string{"n1"}, wrapsFor("m1", "m2", "m2")},
		"empty wrap":         {"key-2", []string{"n1"}, []ChatKeyWrapInput{{UserID: "m1", Wrap: wrap}, {UserID: "m2", Wrap: wrap}, {UserID: "n1"}}},
	}
	for name, tc := range rejected {
		t.Run(name, func(t *testing.T) {
			repo := newRepo()
			svc, token := newAddMembersService(true, true, repo)
			if err := svc.AddMembers(ctx, "chat-1", tc.userIDs, tc.keyID, tc.wraps, token); err == nil {
				t.Fatal("AddMembers() error = nil, want validation error")
			}
			if repo.called {
				t.Fatal("repository was called for an invalid request")
			}
		})
	}

	t.Run("one existing member among several rejects the whole batch", func(t *testing.T) {
		repo := newRepo()
		svc, token := newAddMembersService(true, true, repo)
		err := svc.AddMembers(ctx, "chat-1", []string{"n1", "m2"}, "key-2", wrapsFor("m1", "m2", "n1"), token)
		var already *AlreadyMembersError
		if !errors.As(err, &already) || !slices.Equal(already.UserIDs, []string{"m2"}) || repo.called {
			t.Fatalf("error = %v, called = %v", err, repo.called)
		}
	})

	t.Run("users without an identity key are named", func(t *testing.T) {
		repo := newRepo()
		svc, token := newAddMembersService(true, true, repo)
		err := svc.AddMembers(ctx, "chat-1", []string{"n1", "ghost"}, "key-2", wrapsFor("m1", "m2", "n1", "ghost"), token)
		var missing *MissingIdentityKeysError
		if !errors.As(err, &missing) || !slices.Equal(missing.UserIDs, []string{"ghost"}) || repo.called {
			t.Fatalf("error = %v, called = %v", err, repo.called)
		}
	})

	t.Run("duplicate key id is reported", func(t *testing.T) {
		repo := newRepo()
		repo.err = repository.ErrKeyVersionExists
		svc, token := newAddMembersService(true, true, repo)
		err := svc.AddMembers(ctx, "chat-1", []string{"n1"}, "key-2", wrapsFor("m1", "m2", "n1"), token)
		if !errors.Is(err, ErrKeyVersionExists) {
			t.Fatalf("error = %v, want %v", err, ErrKeyVersionExists)
		}
	})
}
