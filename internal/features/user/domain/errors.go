package domain

import "github.com/jun-develop/junkit/response"

// Domain errors for the user feature.
var (
	ErrUserNotFound       = response.New(response.NotFound, "user_not_found")
	ErrEmailAlreadyExists = response.New(response.Conflict, "email_already_exists")
	ErrInvalidAvatarFile  = response.New(response.BadRequest, "invalid_avatar_file")
	ErrPasswordTooLong    = response.New(response.UnprocessableEntity, "password_too_long")
)
