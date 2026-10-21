package command

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin/binding"

	"github.com/jun-develop/junkit/console"
	"github.com/jun-develop/junkit/validation"

	"jungo/internal/features/user/domain"
	v1dto "jungo/internal/features/user/dto/v1"
)

// CreateUserCommand creates a user (e.g. the first account) with the same validation rules as
// POST /users; the password is read from stdin when -password is omitted.
//
// Usage:
//
//	make console CMD="user:create" ARGS="-email=admin@example.com -first-name=Admin -last-name=User"
type CreateUserCommand struct {
	userService domain.UserService
	validator   *validation.Validator
}

// NewCreateUserCommand creates a CreateUserCommand. Taking the validator also makes Fx construct
// it, which registers the custom rules (password_strong, email_advanced, ...) used below.
func NewCreateUserCommand(userService domain.UserService, validator *validation.Validator) *CreateUserCommand {
	return &CreateUserCommand{userService: userService, validator: validator}
}

// Signature implements console.Command.
func (c *CreateUserCommand) Signature() string {
	return "user:create"
}

// Run implements console.Command.
func (c *CreateUserCommand) Run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet(c.Signature(), flag.ContinueOnError)
	email := flags.String("email", "", "email address (required)")
	firstName := flags.String("first-name", "", "first name (required)")
	lastName := flags.String("last-name", "", "last name (required)")
	password := flags.String("password", "", "password (read from stdin when omitted)")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *password == "" {
		fmt.Print("Password: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return fmt.Errorf("read password: %w", err)
		}

		*password = strings.TrimRight(line, "\r\n")
	}

	req := v1dto.CreateUserRequest{Email: *email, FirstName: *firstName, LastName: *lastName, Password: *password}
	if err := binding.Validator.ValidateStruct(req); err != nil {
		return fmt.Errorf("invalid input: %v", c.validator.GetValidationErrors(nil, err))
	}

	user, err := c.userService.CreateUser(ctx, v1dto.ToCreateUserInput(req))
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	console.Successf("Created user %s <%s>", user.UUID, user.Email)

	return nil
}
