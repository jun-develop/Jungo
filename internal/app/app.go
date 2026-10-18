package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"jungo/internal/config"
)

// App wires together the running HTTP server's dependencies.
type App struct {
	cfg    *config.Config
	router *gin.Engine
	logger *zap.Logger
}

// NewApp assembles an App from its Fx-injected dependencies.
func NewApp(cfg *config.Config, logger *zap.Logger, router *gin.Engine) *App {
	return &App{cfg: cfg, router: router, logger: logger}
}

// Run starts the HTTP server on OnStart and shuts it down gracefully on OnStop.
// The timeouts guard against Slowloris attacks.
func (a *App) Run(lc fx.Lifecycle) {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", a.cfg.ServerPort),
		Handler:           a.router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			a.logger.Info("server: starting",
				zap.String("port", a.cfg.ServerPort),
				zap.String("env", a.cfg.Environment),
			)

			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					a.logger.Error("server: start failed", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			a.logger.Info("server: shutting down")

			if err := srv.Shutdown(ctx); err != nil {
				a.logger.Error("server: graceful shutdown failed", zap.Error(err))
				return err
			}

			a.logger.Info("server: stopped gracefully")

			return nil
		},
	})
}
