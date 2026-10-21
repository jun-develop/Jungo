package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/jun-develop/junkit/database"

	"jungo/internal/database/primary"
	"jungo/internal/database/primary/sqlc"
	"jungo/internal/features/user/domain"
)

// likeEscaper makes a search term match literally inside ILIKE (backslash is the default escape).
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// UserRepository implements domain.UserRepository.
type UserRepository struct {
	db *primary.DB
}

// NewUserRepository creates a UserRepository backed by db.
func NewUserRepository(db *primary.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create implements domain.UserRepository.
func (r *UserRepository) Create(ctx context.Context, input domain.CreateUserInput, passwordHash string) (*domain.User, error) {
	row, err := r.db.Queries(ctx).CreateUser(ctx, sqlc.CreateUserParams{
		Email:        input.Email,
		PasswordHash: passwordHash,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		PhoneNumber:  input.PhoneNumber,
		Status:       domain.UserStatusActive,
	})
	if err != nil {
		return nil, database.Match(err, map[database.ErrorType]error{
			database.ErrorUniqueViolation: domain.ErrEmailAlreadyExists,
		})
	}

	return mapUser(row), nil
}

// GetByUUID implements domain.UserRepository.
func (r *UserRepository) GetByUUID(ctx context.Context, uid uuid.UUID) (*domain.User, error) {
	row, err := r.db.Queries(ctx).GetUserByUUID(ctx, uid)
	if err != nil {
		return nil, database.Match(err, map[database.ErrorType]error{
			database.ErrorNotFound: domain.ErrUserNotFound,
		})
	}

	return mapUser(row), nil
}

// GetByEmail implements domain.UserRepository.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.db.Queries(ctx).GetUserByEmail(ctx, email)
	if err != nil {
		return nil, database.Match(err, map[database.ErrorType]error{
			database.ErrorNotFound: domain.ErrUserNotFound,
		})
	}

	return mapUser(row), nil
}

// GetCredentialsByEmail implements domain.UserRepository.
func (r *UserRepository) GetCredentialsByEmail(ctx context.Context, email string) (*domain.Credentials, error) {
	row, err := r.db.Queries(ctx).GetUserByEmail(ctx, email)
	if err != nil {
		return nil, database.Match(err, map[database.ErrorType]error{
			database.ErrorNotFound: domain.ErrUserNotFound,
		})
	}

	return &domain.Credentials{
		ID:           row.ID,
		UUID:         row.UUID,
		PasswordHash: row.PasswordHash,
		Status:       row.Status,
	}, nil
}

// List implements domain.UserRepository; it also returns the total count for pagination.
func (r *UserRepository) List(ctx context.Context, filter domain.UserListFilter) ([]*domain.User, int64, error) {
	queries := r.db.Queries(ctx)
	search := likeEscaper.Replace(filter.Search)

	rows, err := queries.ListUsers(ctx, sqlc.ListUsersParams{
		Limit:     filter.Limit,
		Offset:    filter.Offset,
		Search:    search,
		SortOrder: filter.SortOrder,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := queries.CountUsers(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	users := make([]*domain.User, len(rows))
	for i, row := range rows {
		users[i] = mapUser(row)
	}

	return users, total, nil
}

// Update implements domain.UserRepository.
func (r *UserRepository) Update(ctx context.Context, uid uuid.UUID, input domain.UpdateUserInput) (*domain.User, error) {
	row, err := r.db.Queries(ctx).UpdateUser(ctx, sqlc.UpdateUserParams{
		UUID:        uid,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		PhoneNumber: input.PhoneNumber,
		AvatarURL:   input.AvatarURL,
		Status:      input.Status,
	})
	if err != nil {
		return nil, database.Match(err, map[database.ErrorType]error{
			database.ErrorNotFound: domain.ErrUserNotFound,
		})
	}

	return mapUser(row), nil
}

// Delete implements domain.UserRepository; it soft-deletes via deleted_at.
func (r *UserRepository) Delete(ctx context.Context, uid uuid.UUID) error {
	rowsAffected, err := r.db.Queries(ctx).DeleteUser(ctx, uid)
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// mapUser converts a generated sqlc.User row to the domain model.
func mapUser(row sqlc.User) *domain.User {
	return &domain.User{
		UUID:        row.UUID,
		Email:       row.Email,
		FirstName:   row.FirstName,
		LastName:    row.LastName,
		PhoneNumber: row.PhoneNumber,
		AvatarURL:   row.AvatarURL,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
