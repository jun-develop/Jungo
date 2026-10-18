package router

import (
	"compress/gzip"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/jun-develop/junkit/middleware"
	"github.com/jun-develop/junkit/notification"
	"github.com/jun-develop/junkit/response"

	"jungo/internal/config"
)

// Routes defines the interface for registering feature routes.
type Routes interface {
	Register(group *gin.RouterGroup)
}

// Versioned defines an API version prefix for routes.
type Versioned interface {
	Version() string
}

// VersionedRoutes combines Routes and Versioned interfaces.
type VersionedRoutes interface {
	Routes
	Versioned
}

// Params defines dependencies for route registration.
type Params struct {
	fx.In

	Router        *gin.Engine
	Config        *config.Config
	Routes        []Routes              `group:"routes"`
	HTTPLogger    *zap.Logger           `name:"http"`
	LimiterLogger *zap.Logger           `name:"limiter"`
	Notifier      notification.Notifier `optional:"true"`
	Responder     response.Responder
}

// RegisterAll sets up global middlewares, system routes, and feature routes.
func RegisterAll(p Params) {
	registerGlobalMiddlewares(p)
	p.Router.HandleMethodNotAllowed = true

	p.Router.NoRoute(func(ctx *gin.Context) {
		p.Responder.Send(ctx, http.StatusNotFound, "api_not_found")
	})
	p.Router.NoMethod(func(ctx *gin.Context) {
		p.Responder.Send(ctx, http.StatusNotFound, "method_not_found")
	})
	p.Router.GET("/health", func(ctx *gin.Context) {
		p.Responder.Send(ctx, http.StatusOK, "app_running")
	})
	p.Router.Static("/uploads", p.Config.Storage.BaseDir)

	api := p.Router.Group("/api")
	versionGroups := make(map[string]*gin.RouterGroup)

	for _, route := range p.Routes {
		target := api

		if v, ok := route.(Versioned); ok && v.Version() != "" {
			version := v.Version()
			group, exists := versionGroups[version]
			if !exists {
				group = api.Group("/" + version)
				versionGroups[version] = group
			}

			target = group
		}

		route.Register(target)
	}
}

// registerGlobalMiddlewares attaches global middlewares to the router.
func registerGlobalMiddlewares(p Params) {
	cfg := p.Config
	debug := middleware.TracerDebugOptions{
		QueryKey:     cfg.TracerDebugKey,
		QueryValue:   cfg.TracerDebugValue,
		SecretHeader: "X-Internal-Secret",
		Secret:       cfg.Auth.InternalSecret,
	}

	p.Router.Use(
		middleware.TracerDebug(debug),
		middleware.Security(),
		middleware.Trace(middleware.TraceOptions{
			Version:     cfg.Version,
			Commit:      cfg.Commit,
			Environment: cfg.Environment,
		}),
		middleware.CORS(middleware.CORSOptions{
			AllowOrigins:     cfg.CORS.AllowOrigins,
			AllowMethods:     cfg.CORS.AllowMethods,
			AllowHeaders:     cfg.CORS.AllowHeaders,
			AllowCredentials: cfg.CORS.AllowCredentials,
			MaxAge:           cfg.CORS.MaxAge,
		}),
		middleware.Gzip(middleware.GzipOptions{Level: gzip.BestSpeed}),
		middleware.Payload(middleware.PayloadOptions{
			MaxBodySize: cfg.Security.MaxRequestBodySize,
		}, p.Responder),
		middleware.AccessLog(p.HTTPLogger, middleware.AccessLogOptions{
			SkipIf: debug.Matches,
		}),
		middleware.Limiter(middleware.LimiterOptions{
			RequestsPerSecond: cfg.Limiter.RequestsPerSecond,
			Burst:             cfg.Limiter.Burst,
			NumShards:         cfg.Limiter.NumShards,
			CleanupInterval:   cfg.Limiter.CleanupInterval,
			ClientTTL:         cfg.Limiter.ClientTTL,
			LogTTL:            cfg.Limiter.LogTTL,
			IPWhitelist:       cfg.Limiter.IPWhitelist,
		}, p.LimiterLogger, p.Responder, p.Notifier),
		middleware.RoutePatterns(map[string]func(string) bool{
			"uuid": middleware.IsUUID,
			"id":   middleware.IsID,
			"slug": middleware.IsSlug,
		}, p.Responder),
		middleware.Recover(p.HTTPLogger, p.Notifier, p.Responder, middleware.RecoverOptions{Debug: debug}),
	)
}
