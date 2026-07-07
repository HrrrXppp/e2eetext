package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

type userRepoStub struct {
	users []domain.User
}

func (s *userRepoStub) List(_ context.Context, filter repository.UserFilter) ([]domain.User, error) {
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

func (s *userRepoStub) Search(_ context.Context, _ repository.UserSearchFilter, _ string) ([]domain.User, error) {
	return nil, nil
}

func (s *userRepoStub) Create(_ context.Context, user domain.User) (domain.User, error) {
	for _, existing := range s.users {
		if existing.OIDCProviderID == user.OIDCProviderID && existing.Subject == user.Subject {
			return domain.User{}, repository.ErrDuplicateUser
		}
	}
	user.ID = "11111111-1111-1111-1111-111111111111"
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = user.CreatedAt
	return user, nil
}

func (s *userRepoStub) GetByID(_ context.Context, id string) (domain.User, error) {
	return domain.User{ID: id}, nil
}

func (s *userRepoStub) UpdateName(_ context.Context, id string, name string) (domain.User, error) {
	for i, user := range s.users {
		if user.ID == id {
			s.users[i].Name = name
			return s.users[i], nil
		}
	}
	return domain.User{ID: id, Name: name}, nil
}

type userOIDCProviderRepoStub struct {
	providers []domain.OIDCProvider
}

func (s *userOIDCProviderRepoStub) List(_ context.Context) ([]domain.OIDCProvider, error) {
	return s.providers, nil
}

func (s *userOIDCProviderRepoStub) GetByName(_ context.Context, name string) (domain.OIDCProvider, error) {
	for _, provider := range s.providers {
		if provider.Name == name {
			return provider, nil
		}
	}
	return domain.OIDCProvider{}, errors.New("oidc provider not found")
}

func newUserService(repo *userRepoStub, oidc ...*userOIDCProviderRepoStub) *UserService {
	providerRepo := &userOIDCProviderRepoStub{}
	if len(oidc) > 0 && oidc[0] != nil {
		providerRepo = oidc[0]
	}
	return NewUserService(repo, providerRepo)
}

func TestUserService_List_Validation(t *testing.T) {
	svc := newUserService(&userRepoStub{})

	_, err := svc.List(context.Background(), repository.UserFilter{
		Subject: "subject-only",
	})
	if err == nil || err.Error() != "oidc_provider_id is required" {
		t.Fatalf("List() error = %v", err)
	}
}

func TestUserService_Create_Validation(t *testing.T) {
	svc := newUserService(&userRepoStub{})

	_, err := svc.Create(context.Background(), CreateUserInput{
		Subject: "subject-only",
	})
	if err == nil || err.Error() != "oidc_provider_id is required" {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.Create(context.Background(), CreateUserInput{
		Subject:        "subject-only",
		OIDCProviderID: "provider-id",
	})
	if err == nil || err.Error() != "kem_public_key is required" {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestUserService_Create_Duplicate(t *testing.T) {
	svc := newUserService(&userRepoStub{
		users: []domain.User{
			{OIDCProviderID: "provider-id", Subject: "subject"},
		},
	})

	_, err := svc.Create(context.Background(), CreateUserInput{
		Subject:        "subject",
		OIDCProviderID: "provider-id",
		KemPublicKey:   []byte("user-kem-public-key"),
	})
	if err == nil {
		t.Fatal("Create() error = nil, want duplicate error")
	}
	if !errors.Is(err, repository.ErrDuplicateUser) {
		t.Fatalf("Create() error = %v, want duplicate user", err)
	}
}

func TestUserService_CreateFromToken(t *testing.T) {
	svc := newUserService(&userRepoStub{}, &userOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: "provider-id", Name: "google"},
		},
	})

	user, err := svc.CreateFromToken(context.Background(), TokenUser{
		Subject:  "google-subject-1",
		Name:     "Alice",
		Provider: "google",
	}, false, []byte("user-kem-public-key"))
	if err != nil {
		t.Fatalf("CreateFromToken() error = %v", err)
	}
	if user.Subject != "google-subject-1" {
		t.Errorf("Subject = %q", user.Subject)
	}
	if user.Name != "Alice" {
		t.Errorf("Name = %q", user.Name)
	}
}

func TestUserService_CreateFromToken_SkipProfile(t *testing.T) {
	svc := newUserService(&userRepoStub{}, &userOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: "provider-id", Name: "google"},
		},
	})

	user, err := svc.CreateFromToken(context.Background(), TokenUser{
		Subject:  "google-subject-1",
		Name:     "Alice",
		Provider: "google",
	}, true, []byte("user-kem-public-key"))
	if err != nil {
		t.Fatalf("CreateFromToken() error = %v", err)
	}
	if user.Name != "" {
		t.Errorf("Name = %q, want empty", user.Name)
	}
}

func TestUserService_SearchFromQuery(t *testing.T) {
	repo := &searchUserRepoStub{}
	svc := NewUserService(repo, &userOIDCProviderRepoStub{})

	_, err := svc.Search(context.Background(), "ali", "99999999-9999-9999-9999-999999999999")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if repo.lastFilter.NameQuery != "ali" {
		t.Fatalf("NameQuery = %q", repo.lastFilter.NameQuery)
	}
	if repo.lastFilter.UserIDQuery != "" {
		t.Fatalf("UserIDQuery = %q, want empty", repo.lastFilter.UserIDQuery)
	}

	_, err = svc.Search(context.Background(), "12345", "99999999-9999-9999-9999-999999999999")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if repo.lastFilter.UserIDQuery != "12345" {
		t.Fatalf("UserIDQuery = %q", repo.lastFilter.UserIDQuery)
	}

	users, err := svc.Search(context.Background(), "Евг", "99999999-9999-9999-9999-999999999999")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("Search() len = %d, want 0", len(users))
	}
	if repo.lastFilter.NameQuery != "Евг" {
		t.Fatalf("NameQuery = %q", repo.lastFilter.NameQuery)
	}

	users, err = svc.Search(context.Background(), "Ев", "99999999-9999-9999-9999-999999999999")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("Search() len = %d, want 0 for short query", len(users))
	}
}

type searchUserRepoStub struct {
	userRepoStub
	lastFilter repository.UserSearchFilter
}

func (s *searchUserRepoStub) Search(_ context.Context, filter repository.UserSearchFilter, _ string) ([]domain.User, error) {
	s.lastFilter = filter
	return nil, nil
}

func TestUserService_UpdateName_OwnProfile(t *testing.T) {
	const (
		userID     = "11111111-1111-1111-1111-111111111111"
		providerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	)

	repo := &userRepoStub{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
				Name:           "Old Name",
			},
		},
	}
	svc := newUserService(repo, &userOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "google"},
		},
	})

	user, err := svc.UpdateName(context.Background(), userID, "  New Name  ", TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if err != nil {
		t.Fatalf("UpdateName() error = %v", err)
	}
	if user.Name != "New Name" {
		t.Errorf("Name = %q, want %q", user.Name, "New Name")
	}
}

func TestUserService_UpdateName_DeniedForOtherUser(t *testing.T) {
	const (
		userID     = "11111111-1111-1111-1111-111111111111"
		otherID    = "22222222-2222-2222-2222-222222222222"
		providerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	)

	repo := &userRepoStub{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
			},
		},
	}
	svc := newUserService(repo, &userOIDCProviderRepoStub{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "google"},
		},
	})

	_, err := svc.UpdateName(context.Background(), otherID, "Hacked", TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	if !errors.Is(err, ErrUserAccessDenied) {
		t.Fatalf("UpdateName() error = %v, want %v", err, ErrUserAccessDenied)
	}
}
