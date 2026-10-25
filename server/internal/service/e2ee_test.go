package service

import (
	"context"
	"encoding/json"
	"errors"
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
