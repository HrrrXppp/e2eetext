package repository

import (
	"context"
	"errors"

	"github.com/ekhrunov/messenger/server/internal/domain"
)

var ErrDuplicateUser = errors.New("user already exists")

type UserFilter struct {
	Subject        string
	OIDCProviderID string
	Name           string
}

type UserSearchFilter struct {
	NameQuery   string
	UserIDQuery string
	Limit       int
}

type UserRepository interface {
	List(ctx context.Context, filter UserFilter) ([]domain.User, error)
	Search(ctx context.Context, filter UserSearchFilter, nodeID string) ([]domain.User, error)
	Create(ctx context.Context, user domain.User) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
	UpdateName(ctx context.Context, id string, name string) (domain.User, error)
}
