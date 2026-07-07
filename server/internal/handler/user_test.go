package handler

import (
	"bytes"
	"context"
	"encoding/base64"
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

type stubUserRepo struct {
	users []domain.User
}

func (s *stubUserRepo) List(_ context.Context, filter repository.UserFilter) ([]domain.User, error) {
	var users []domain.User
	for _, user := range s.users {
		if filter.Subject != "" && user.Subject != filter.Subject {
			continue
		}
		if filter.OIDCProviderID != "" && user.OIDCProviderID != filter.OIDCProviderID {
			continue
		}
		if filter.Name != "" && user.Name != filter.Name {
			continue
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *stubUserRepo) Search(_ context.Context, filter repository.UserSearchFilter, nodeID string) ([]domain.User, error) {
	var users []domain.User
	for _, user := range s.users {
		if filter.NameQuery != "" && strings.Contains(strings.ToLower(user.Name), strings.ToLower(filter.NameQuery)) {
			users = append(users, user)
			continue
		}
		if filter.UserIDQuery != "" {
			scopedID := user.ID
			if nodeID != "" {
				scopedID = nodeID + "/" + user.ID
			}
			if strings.Contains(user.ID, filter.UserIDQuery) || strings.Contains(scopedID, filter.UserIDQuery) {
				users = append(users, user)
			}
		}
	}
	return users, nil
}

func (s *stubUserRepo) Create(_ context.Context, user domain.User) (domain.User, error) {
	for _, existing := range s.users {
		if existing.OIDCProviderID == user.OIDCProviderID && existing.Subject == user.Subject {
			return domain.User{}, repository.ErrDuplicateUser
		}
	}

	user.ID = "33333333-3333-3333-3333-333333333333"
	user.CreatedAt = time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	user.UpdatedAt = user.CreatedAt
	s.users = append(s.users, user)
	return user, nil
}

func (s *stubUserRepo) GetByID(_ context.Context, id string) (domain.User, error) {
	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}
	return domain.User{}, errors.New("user not found")
}

func (s *stubUserRepo) UpdateName(_ context.Context, id string, name string) (domain.User, error) {
	for i, user := range s.users {
		if user.ID == id {
			s.users[i].Name = name
			s.users[i].UpdatedAt = time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
			return s.users[i], nil
		}
	}
	return domain.User{}, errors.New("user not found")
}

type stubOIDCProviderRepo struct {
	providers []domain.OIDCProvider
}

func (s *stubOIDCProviderRepo) List(_ context.Context) ([]domain.OIDCProvider, error) {
	return s.providers, nil
}

func (s *stubOIDCProviderRepo) GetByName(_ context.Context, name string) (domain.OIDCProvider, error) {
	for _, provider := range s.providers {
		if provider.Name == name {
			return provider, nil
		}
	}
	return domain.OIDCProvider{}, errors.New("oidc provider not found")
}

func newUserHandler(repo *stubUserRepo, oidc ...*stubOIDCProviderRepo) *UserHandler {
	providerRepo := &stubOIDCProviderRepo{}
	if len(oidc) > 0 && oidc[0] != nil {
		providerRepo = oidc[0]
	}
	return NewUserHandler(service.NewUserService(repo, providerRepo), testNode())
}

func testOIDCProviderRepo() *stubOIDCProviderRepo {
	return &stubOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{
				ID:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Name: "google",
			},
		},
	}
}

func TestUserHandler_List(t *testing.T) {
	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             "11111111-1111-1111-1111-111111111111",
				OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Subject:        "google-subject-1",
				Name:           "Alice",
			},
			{
				ID:             "22222222-2222-2222-2222-222222222222",
				OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Subject:        "google-subject-2",
				Name:           "Bob",
			},
		},
	}
	handler := newUserHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user?subject=google-subject-1&oidc_provider_id=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var users []domain.User
	if err := json.NewDecoder(rec.Body).Decode(&users); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("users len = %d, want 1", len(users))
	}
	if users[0].Subject != "google-subject-1" {
		t.Errorf("Subject = %q", users[0].Subject)
	}
}

func TestUserHandler_List_WithNameFilter(t *testing.T) {
	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             "11111111-1111-1111-1111-111111111111",
				OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Subject:        "google-subject-1",
				Name:           "Alice",
			},
			{
				ID:             "22222222-2222-2222-2222-222222222222",
				OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Subject:        "google-subject-2",
				Name:           "Bob",
			},
		},
	}
	handler := newUserHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user?subject=google-subject-1&oidc_provider_id=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa&name=Alice", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var users []domain.User
	if err := json.NewDecoder(rec.Body).Decode(&users); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("users len = %d, want 1", len(users))
	}
	if users[0].Name != "Alice" {
		t.Errorf("Name = %q", users[0].Name)
	}
}

func TestUserHandler_Search(t *testing.T) {
	testUserHandlerSearch(t, "/api/v1/search?q=ali")
}

func testUserHandlerSearch(t *testing.T, path string) {
	t.Helper()

	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             "11111111-1111-1111-1111-111111111111",
				OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Subject:        "google-subject-1",
				Name:           "Alice",
			},
			{
				ID:             "22222222-2222-2222-2222-222222222222",
				OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Subject:        "google-subject-2",
				Name:           "Bob",
			},
		},
	}
	handler := newUserHandler(repo)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var users []domain.User
	if err := json.NewDecoder(rec.Body).Decode(&users); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("users len = %d, want 1", len(users))
	}
	if users[0].Name != "Alice" {
		t.Errorf("Name = %q", users[0].Name)
	}
}

func TestUserHandler_Search_CyrillicName(t *testing.T) {
	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             "11111111-1111-1111-1111-111111111111",
				OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Subject:        "google-subject-1",
				Name:           "Евгений Хрунов",
			},
		},
	}
	handler := newUserHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=%D0%95%D0%B2%D0%B3", nil)
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var users []domain.User
	if err := json.NewDecoder(rec.Body).Decode(&users); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("users len = %d, want 1", len(users))
	}
	if users[0].Name != "Евгений Хрунов" {
		t.Errorf("Name = %q", users[0].Name)
	}
}

func TestUserHandler_Search_ShortQueryReturnsEmptyList(t *testing.T) {
	handler := newUserHandler(&stubUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=ab", nil)
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var users []domain.User
	if err := json.NewDecoder(rec.Body).Decode(&users); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("users len = %d, want 0", len(users))
	}
}

func TestUserHandler_List_MissingFilters(t *testing.T) {
	handler := newUserHandler(&stubUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user?subject=google-subject-1", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_Create(t *testing.T) {
	repo := &stubUserRepo{}
	handler := newUserHandler(repo, testOIDCProviderRepo())

	body := []byte(`{"kem_public_key":"` + base64.StdEncoding.EncodeToString([]byte("user-kem-public-key")) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user", bytes.NewReader(body))
	req = requestWithTokenUser(req, service.TokenUser{
		Subject:  "google-subject-1",
		Name:     "Test User",
		Provider: "google",
	})
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var user domain.User
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.Subject != "google-subject-1" {
		t.Errorf("Subject = %q", user.Subject)
	}
	if user.Name != "Test User" {
		t.Errorf("Name = %q", user.Name)
	}
}

func TestUserHandler_Create_UnauthorizedWithoutTokenUser(t *testing.T) {
	handler := newUserHandler(&stubUserRepo{}, testOIDCProviderRepo())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user", nil)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestUserHandler_Create_RejectsIdentityFieldsInBody(t *testing.T) {
	handler := newUserHandler(&stubUserRepo{}, testOIDCProviderRepo())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user", bytes.NewReader([]byte(`{"subject":"x"}`)))
	req = requestWithTokenUser(req, service.TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_Create_ValidationError(t *testing.T) {
	handler := newUserHandler(&stubUserRepo{}, testOIDCProviderRepo())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user", nil)
	req = requestWithTokenUser(req, service.TokenUser{
		Subject:  "google-subject-1",
		Provider: "unknown",
	})
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_Create_SkipProfile(t *testing.T) {
	repo := &stubUserRepo{}
	handler := newUserHandler(repo, testOIDCProviderRepo())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user", bytes.NewReader([]byte(`{"skip_profile":true,"kem_public_key":"`+base64.StdEncoding.EncodeToString([]byte("user-kem-public-key"))+`"}`)))
	req = requestWithTokenUser(req, service.TokenUser{
		Subject:  "google-subject-1",
		Name:     "Test User",
		Provider: "google",
	})
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var user domain.User
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.Name != "" {
		t.Errorf("Name = %q, want empty", user.Name)
	}
}

func TestUserHandler_Create_Conflict(t *testing.T) {
	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             "11111111-1111-1111-1111-111111111111",
				OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Subject:        "google-subject-1",
			},
		},
	}
	handler := newUserHandler(repo, testOIDCProviderRepo())

	body := []byte(`{"kem_public_key":"` + base64.StdEncoding.EncodeToString([]byte("user-kem-public-key")) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user", bytes.NewReader(body))
	req = requestWithTokenUser(req, service.TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestUserHandler_GetByID(t *testing.T) {
	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             "11111111-1111-1111-1111-111111111111",
				OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Subject:        "google-subject-1",
				Name:           "Test User",
			},
		},
	}
	handler := newUserHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/11111111-1111-1111-1111-111111111111", nil)
	req.SetPathValue("id", "11111111-1111-1111-1111-111111111111")
	rec := httptest.NewRecorder()

	handler.GetByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var user domain.User
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.Name != "Test User" {
		t.Errorf("Name = %q", user.Name)
	}
}

func TestUserHandler_GetByID_NotFound(t *testing.T) {
	handler := newUserHandler(&stubUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/missing", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	handler.GetByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUserHandler_Update(t *testing.T) {
	const (
		userID     = "11111111-1111-1111-1111-111111111111"
		providerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	)

	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
				Name:           "Old Name",
			},
		},
	}
	handler := newUserHandler(repo, &stubOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "google"},
		},
	})

	body := []byte(`{"name":"New Name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/"+userID, bytes.NewReader(body))
	req.SetPathValue("id", userID)
	req = req.WithContext(context.WithValue(req.Context(), service.TokenUserContextKey, service.TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	}))
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var user domain.User
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.Name != "New Name" {
		t.Errorf("Name = %q, want %q", user.Name, "New Name")
	}
}

func TestUserHandler_Update_ScopedID(t *testing.T) {
	const (
		nodeID     = "99999999-9999-9999-9999-999999999999"
		userID     = "11111111-1111-1111-1111-111111111111"
		providerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	)

	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
				Name:           "Old Name",
			},
		},
	}
	handler := newUserHandler(repo, &stubOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "google"},
		},
	})

	scopedID := nodeID + "/" + userID
	body := []byte(`{"name":"New Name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/"+scopedID, bytes.NewReader(body))
	req.SetPathValue("nodeId", nodeID)
	req.SetPathValue("localId", userID)
	req = req.WithContext(context.WithValue(req.Context(), service.TokenUserContextKey, service.TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	}))
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var user domain.User
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.Name != "New Name" {
		t.Errorf("Name = %q, want %q", user.Name, "New Name")
	}
}

func TestUserHandler_Update_ForbiddenForOtherUser(t *testing.T) {
	const (
		userID     = "11111111-1111-1111-1111-111111111111"
		otherID    = "22222222-2222-2222-2222-222222222222"
		providerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	)

	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
				Name:           "Alice",
			},
			{
				ID:             otherID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-2",
				Name:           "Bob",
			},
		},
	}
	handler := newUserHandler(repo, &stubOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "google"},
		},
	})

	body := []byte(`{"name":"Hacked"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/"+otherID, bytes.NewReader(body))
	req.SetPathValue("id", otherID)
	req = req.WithContext(context.WithValue(req.Context(), service.TokenUserContextKey, service.TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	}))
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestUserHandler_Update_ForbiddenForOtherUser_ScopedID(t *testing.T) {
	const (
		userID     = "11111111-1111-1111-1111-111111111111"
		otherID    = "22222222-2222-2222-2222-222222222222"
		providerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	)

	repo := &stubUserRepo{
		users: []domain.User{
			{
				ID:             userID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-1",
				Name:           "Alice",
			},
			{
				ID:             otherID,
				OIDCProviderID: providerID,
				Subject:        "google-subject-2",
				Name:           "Bob",
			},
		},
	}
	handler := newUserHandler(repo, &stubOIDCProviderRepo{
		providers: []domain.OIDCProvider{
			{ID: providerID, Name: "google"},
		},
	})

	scopedOtherID := scopedID(otherID)
	body := []byte(`{"name":"Hacked"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/"+scopedOtherID, bytes.NewReader(body))
	req.SetPathValue("nodeId", testNodeID)
	req.SetPathValue("localId", otherID)
	req = requestWithTokenUser(req, service.TokenUser{
		Subject:  "google-subject-1",
		Provider: "google",
	})
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	for _, user := range repo.users {
		if user.ID == otherID && user.Name != "Bob" {
			t.Errorf("other user name = %q, want %q", user.Name, "Bob")
		}
	}
}
