package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
	"github.com/lib/pq"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

var _ repository.UserRepository = (*UserRepository)(nil)

func (r *UserRepository) List(ctx context.Context, filter repository.UserFilter) ([]domain.User, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, oidc_provider_id, subject, name, created_at, updated_at
		FROM users
		WHERE 1=1`)

	args := make([]any, 0, 3)
	if filter.Subject != "" {
		args = append(args, filter.Subject)
		fmt.Fprintf(&query, " AND subject = $%d", len(args))
	}
	if filter.OIDCProviderID != "" {
		args = append(args, filter.OIDCProviderID)
		fmt.Fprintf(&query, " AND oidc_provider_id = $%d", len(args))
	}
	if filter.Name != "" {
		args = append(args, filter.Name)
		fmt.Fprintf(&query, " AND name = $%d", len(args))
	}

	query.WriteString(" ORDER BY created_at DESC")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

func (r *UserRepository) Search(ctx context.Context, filter repository.UserSearchFilter, nodeID string) ([]domain.User, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, oidc_provider_id, subject, name, created_at, updated_at
		FROM users`)

	args := make([]any, 0, 4)
	var conditions []string

	if filter.NameQuery != "" {
		args = append(args, "%"+filter.NameQuery+"%")
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	if filter.UserIDQuery != "" {
		args = append(args, "%"+filter.UserIDQuery+"%")
		idCondition := fmt.Sprintf("id::text ILIKE $%d", len(args))
		if nodeID != "" {
			args = append(args, nodeID)
			idCondition = fmt.Sprintf("(%s OR ($%d || '/' || id::text) ILIKE $%d)", idCondition, len(args), len(args)-1)
		}
		conditions = append(conditions, idCondition)
	}

	if len(conditions) == 0 {
		return []domain.User{}, nil
	}

	query.WriteString(" WHERE (")
	query.WriteString(strings.Join(conditions, " OR "))
	query.WriteString(") ORDER BY name NULLS LAST, created_at DESC")

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	args = append(args, limit)
	fmt.Fprintf(&query, " LIMIT $%d", len(args))

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search users: %w", err)
	}

	return users, nil
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO users (oidc_provider_id, subject, name)
		VALUES ($1, $2, $3)
		RETURNING id, oidc_provider_id, subject, name, created_at, updated_at`,
		user.OIDCProviderID,
		user.Subject,
		nullString(user.Name),
	)

	created, err := scanUser(row)
	if err != nil {
		if isDuplicateUserError(err) {
			return domain.User{}, repository.ErrDuplicateUser
		}
		return domain.User{}, err
	}

	return created, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, oidc_provider_id, subject, name, created_at, updated_at
		FROM users
		WHERE id = $1`, id)

	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (r *UserRepository) UpdateName(ctx context.Context, id string, name string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE users
		SET name = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, oidc_provider_id, subject, name, created_at, updated_at`,
		nullString(name),
		id,
	)

	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

type userRow interface {
	Scan(dest ...any) error
}

func scanUser(row userRow) (domain.User, error) {
	var user domain.User
	var name sql.NullString

	if err := row.Scan(
		&user.ID,
		&user.OIDCProviderID,
		&user.Subject,
		&name,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, fmt.Errorf("user not found")
		}
		return domain.User{}, fmt.Errorf("scan user: %w", err)
	}

	user.Name = name.String

	return user, nil
}

func isDuplicateUserError(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
