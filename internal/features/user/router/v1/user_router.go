package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jun-develop/junkit/middleware"
	"github.com/jun-develop/junkit/response"

	authdomain "jungo/internal/features/auth/domain"
	v1handler "jungo/internal/features/user/handler/v1"
	"jungo/internal/router"
)

// UserRoutes registers the user feature's endpoints under a version group.
type UserRoutes struct {
	handler       *v1handler.UserHandler
	authenticator middleware.Authenticator[*authdomain.Identity]
	responder     response.Responder
}

// NewUserRoutes creates a UserRoutes.
func NewUserRoutes(handler *v1handler.UserHandler, authenticator middleware.Authenticator[*authdomain.Identity], responder response.Responder) *UserRoutes {
	return &UserRoutes{handler: handler, authenticator: authenticator, responder: responder}
}

// Version implements router.Versioned; routes are nested under "/api/v1".
func (r *UserRoutes) Version() string {
	return "v1"
}

// Register implements router.Routes; every route requires a logged-in user.
func (r *UserRoutes) Register(group *gin.RouterGroup) {
	users := group.Group("/users")
	users.Use(middleware.BearerAuth(r.authenticator, r.responder, middleware.BearerAuthOptions{}))

	users.POST("", r.handler.CreateUser)
	users.GET("", r.handler.ListUsers)
	users.GET("/:uuid", r.handler.GetUser)
	users.PATCH("/:uuid", r.handler.UpdateUser)
	users.DELETE("/:uuid", r.handler.DeleteUser)
	users.POST("/:uuid/avatar", r.handler.UploadAvatar)
	users.DELETE("/:uuid/avatar", r.handler.DeleteAvatar)
}

// multipartOverhead is the room left in a body limit for multipart boundaries and part headers.
const multipartOverhead = 64 << 10

// BodyLimits implements router.BodyLimited; the avatar upload is larger than SECURITY_MAX_BODY_SIZE.
func (r *UserRoutes) BodyLimits() []router.BodyLimit {
	return []router.BodyLimit{
		{
			Method: http.MethodPost, 
			Path: "/users/:uuid/avatar", 
			MaxBytes: v1handler.MaxAvatarFileSize + multipartOverhead,
		},
	}
}
