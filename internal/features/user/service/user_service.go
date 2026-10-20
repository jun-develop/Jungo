package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/jun-develop/junkit/security"
	"github.com/jun-develop/junkit/storage"
	"github.com/jun-develop/junkit/tracer"

	"jungo/internal/features/user/domain"
)

// UserService implements domain.UserService.
type UserService struct {
	repo     domain.UserRepository
	storage  storage.Service
	hasher   security.PasswordHasher
	sessions domain.SessionRevoker
}

// NewUserService creates a UserService.
func NewUserService(repo domain.UserRepository, storageService storage.Service, hasher security.PasswordHasher, sessions domain.SessionRevoker) *UserService {
	return &UserService{repo: repo, storage: storageService, hasher: hasher, sessions: sessions}
}

// CreateUser creates a new user with hashed password.
//
// Flow: normalize email -> hash password -> persist user
func (s *UserService) CreateUser(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	input.Email = domain.NormalizeEmail(input.Email)

	stop := tracer.Span(ctx, "Bcrypt Hashing")
	passwordHash, err := s.hasher.Hash(input.Password)
	stop()
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, input, passwordHash)
}

// GetUser returns a user by UUID.
func (s *UserService) GetUser(ctx context.Context, uid uuid.UUID) (*domain.User, error) {
	return s.repo.GetByUUID(ctx, uid)
}

// GetCredentialsByEmail returns the login credentials of the user with email.
func (s *UserService) GetCredentialsByEmail(ctx context.Context, email string) (*domain.Credentials, error) {
	return s.repo.GetCredentialsByEmail(ctx, domain.NormalizeEmail(email))
}

// GetUsers returns a paginated list of users.
func (s *UserService) GetUsers(ctx context.Context, filter domain.UserListFilter) ([]*domain.User, int64, error) {
	return s.repo.List(ctx, filter)
}

// UpdateUser updates user information.
//
// Flow: update in DB -> revoke sessions if no longer active
func (s *UserService) UpdateUser(ctx context.Context, uid uuid.UUID, input domain.UpdateUserInput) (*domain.User, error) {
	user, err := s.repo.Update(ctx, uid, input)
	if err != nil {
		return nil, err
	}

	if !user.IsActive() {
		if err := s.revokeSessions(ctx, uid); err != nil {
			return nil, err
		}
	}

	return user, nil
}

// DeleteUser deletes a user by UUID.
//
// Flow: soft-delete in DB -> revoke sessions
func (s *UserService) DeleteUser(ctx context.Context, uid uuid.UUID) error {
	if err := s.repo.Delete(ctx, uid); err != nil {
		return err
	}

	return s.revokeSessions(ctx, uid)
}

// UploadAvatar uploads a new avatar and updates the user record.
//
// Flow: get user -> upload file -> update avatar URL -> delete old file
func (s *UserService) UploadAvatar(ctx context.Context, uid uuid.UUID, file domain.AvatarFile) (*domain.User, error) {
	user, err := s.repo.GetByUUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	stop := tracer.Span(ctx, "Upload Avatar to Storage")
	newURL, err := s.storage.UploadFile(ctx, file.Reader, file.Filename)
	stop()
	if err != nil {
		return nil, domain.ErrInvalidAvatarFile
	}

	updated, err := s.repo.Update(ctx, uid, domain.UpdateUserInput{AvatarURL: &newURL})
	if err != nil {
		_ = s.storage.DeleteFile(ctx, newURL)
		return nil, err
	}

	if hasAvatar(user) {
		_ = s.storage.DeleteFile(ctx, *user.AvatarURL)
	}

	return updated, nil
}

// DeleteAvatar removes the avatar file and clears the avatar URL.
//
// Flow: get user -> delete file -> clear avatar URL in DB
func (s *UserService) DeleteAvatar(ctx context.Context, uid uuid.UUID) (*domain.User, error) {
	user, err := s.repo.GetByUUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	if !hasAvatar(user) {
		return user, nil
	}

	_ = s.storage.DeleteFile(ctx, *user.AvatarURL)

	empty := ""

	return s.repo.Update(ctx, uid, domain.UpdateUserInput{AvatarURL: &empty})
}

// hasAvatar checks if the user has an avatar URL.
func hasAvatar(user *domain.User) bool {
	return user.AvatarURL != nil && *user.AvatarURL != ""
}

// revokeSessions ends the user's sessions. Its error is returned: until revoked, a cached session
// keeps the user's old active status for up to AUTH_SESSION_CACHE_TTL.
func (s *UserService) revokeSessions(ctx context.Context, uid uuid.UUID) error {
	if s.sessions == nil {
		return nil
	}

	return s.sessions.RevokeUserSessions(ctx, uid)
}
