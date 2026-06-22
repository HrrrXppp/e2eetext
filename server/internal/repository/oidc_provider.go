package repository

import (
	"context"

	"github.com/ekhrunov/messenger/server/internal/domain"
)

type OIDCProviderRepository interface {
	List(ctx context.Context) ([]domain.OIDCProvider, error)
	GetByName(ctx context.Context, name string) (domain.OIDCProvider, error)
}
