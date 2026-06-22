package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

var ErrUserAccessDenied = errors.New("user access denied")

type UserService struct {
	repo             repository.UserRepository
	oidcProviderRepo repository.OIDCProviderRepository
}

func NewUserService(repo repository.UserRepository, oidcProviderRepo repository.OIDCProviderRepository) *UserService {
	return &UserService{repo: repo, oidcProviderRepo: oidcProviderRepo}
}

type CreateUserInput struct {
	Subject        string
	OIDCProviderID string
	Name           string
}

func (s *UserService) List(ctx context.Context, filter repository.UserFilter) ([]domain.User, error) {
	if filter.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	if filter.OIDCProviderID == "" {
		return nil, fmt.Errorf("oidc_provider_id is required")
	}

	return s.repo.List(ctx, filter)
}

const (
	minUserNameSearchLen = 3
	minUserIDSearchLen   = 5
)

func (s *UserService) Search(ctx context.Context, query string, nodeID string) ([]domain.User, error) {
	query = strings.TrimSpace(query)
	if query == "" || utf8.RuneCountInString(query) < minUserNameSearchLen {
		return []domain.User{}, nil
	}

	filter := repository.UserSearchFilter{
		NameQuery: query,
		Limit:     20,
	}
	if utf8.RuneCountInString(query) >= minUserIDSearchLen {
		filter.UserIDQuery = query
	}

	return s.repo.Search(ctx, filter, nodeID)
}

func (s *UserService) Create(ctx context.Context, input CreateUserInput) (domain.User, error) {
	if input.Subject == "" {
		return domain.User{}, fmt.Errorf("subject is required")
	}
	if input.OIDCProviderID == "" {
		return domain.User{}, fmt.Errorf("oidc_provider_id is required")
	}

	user, err := s.repo.Create(ctx, domain.User{
		OIDCProviderID: input.OIDCProviderID,
		Subject:        input.Subject,
		Name:           input.Name,
	})
	if errors.Is(err, repository.ErrDuplicateUser) {
		return domain.User{}, repository.ErrDuplicateUser
	}
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (s *UserService) CreateFromToken(ctx context.Context, tokenUser TokenUser, skipProfile bool) (domain.User, error) {
	if tokenUser.Subject == "" {
		return domain.User{}, fmt.Errorf("subject is required")
	}
	if tokenUser.Provider == "" {
		return domain.User{}, fmt.Errorf("provider is required")
	}

	provider, err := s.oidcProviderRepo.GetByName(ctx, tokenUser.Provider)
	if err != nil {
		return domain.User{}, fmt.Errorf("oidc provider not found")
	}

	name := ""
	if !skipProfile {
		name = strings.TrimSpace(tokenUser.Name)
	}

	return s.Create(ctx, CreateUserInput{
		Subject:        tokenUser.Subject,
		OIDCProviderID: provider.ID,
		Name:           name,
	})
}

func (s *UserService) GetByID(ctx context.Context, id string) (domain.User, error) {
	if id == "" {
		return domain.User{}, fmt.Errorf("id is required")
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (s *UserService) UpdateName(ctx context.Context, id string, name string, tokenUser TokenUser) (domain.User, error) {
	if id == "" {
		return domain.User{}, fmt.Errorf("id is required")
	}

	if err := requireCurrentUserMatches(ctx, tokenUser, id, s.repo, s.oidcProviderRepo, ErrUserAccessDenied); err != nil {
		return domain.User{}, err
	}

	return s.repo.UpdateName(ctx, id, strings.TrimSpace(name))
}
