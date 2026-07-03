package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

type OIDCProviderRepository struct {
	db *sql.DB
}

func NewOIDCProviderRepository(db *sql.DB) *OIDCProviderRepository {
	return &OIDCProviderRepository{db: db}
}

var _ repository.OIDCProviderRepository = (*OIDCProviderRepository)(nil)

func (r *OIDCProviderRepository) List(ctx context.Context) ([]domain.OIDCProvider, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, link, picture
		FROM oidc_providers
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list oidc providers: %w", err)
	}
	defer rows.Close()

	var providers []domain.OIDCProvider
	for rows.Next() {
		provider, err := scanOIDCProvider(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate oidc providers: %w", err)
	}

	return providers, nil
}

func (r *OIDCProviderRepository) GetByName(ctx context.Context, name string) (domain.OIDCProvider, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, link, picture
		FROM oidc_providers
		WHERE lower(name) = lower($1)`, name)

	provider, err := scanOIDCProvider(row)
	if err != nil {
		return domain.OIDCProvider{}, err
	}

	return provider, nil
}

type oidcProviderRow interface {
	Scan(dest ...any) error
}

func scanOIDCProvider(row oidcProviderRow) (domain.OIDCProvider, error) {
	var provider domain.OIDCProvider
	var picture []byte

	if err := row.Scan(&provider.ID, &provider.Name, &provider.Link, &picture); err != nil {
		if err == sql.ErrNoRows {
			return domain.OIDCProvider{}, fmt.Errorf("oidc provider not found")
		}
		return domain.OIDCProvider{}, fmt.Errorf("scan oidc provider: %w", err)
	}

	if len(picture) > 0 {
		provider.Picture = picture
	}

	return provider, nil
}
