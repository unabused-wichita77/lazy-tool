package app

import (
	"context"
	"log/slog"

	"lazy-tool/internal/config"
)

// App wires shared runtime state for CLI commands (expanded in later phases).
type App struct {
	Config *config.Config
	Log    *slog.Logger
}

// New returns an App with the given config and logger.
func New(cfg *config.Config, log *slog.Logger) *App {
	return &App{Config: cfg, Log: log}
}

// Run logs bootstrap info. Long-running commands (e.g. serve) will wait on ctx.
func (a *App) Run(ctx context.Context) error {
	if a.Config != nil && a.Config.App.Name != "" {
		a.Log.Info("lazy-tool ready", "app", a.Config.App.Name, "env", a.Config.App.Environment)
	} else {
		a.Log.Info("lazy-tool ready")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
