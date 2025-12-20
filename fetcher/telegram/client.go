package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	tdauth "github.com/gotd/td/telegram/auth"
)

// ClientRunner is a function that runs with an authenticated client
type ClientRunner func(ctx context.Context, client *telegram.Client) error

// RunWithAuth creates a Telegram client, authenticates it, and runs the provided function
func RunWithAuth(ctx context.Context, configDir string, appID int, appHash string, phoneNumber string, runner ClientRunner) error {

	// Set up session storage
	sessionPath := filepath.Join(configDir, "telegram-session.json")
	sessionStorage := &session.FileStorage{
		Path: sessionPath,
	}

	// Set up flood wait handler
	waiter := floodwait.NewWaiter().WithCallback(func(ctx context.Context, wait floodwait.FloodWait) {
		slog.Warn("telegram rate limit", "retry_after", wait.Duration)
	})

	// Create client with logger to see connection issues
	slog.Info("creating telegram client")

	// Create zap logger to see gotd internal logs
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger, _ := config.Build()

	client := telegram.NewClient(appID, appHash, telegram.Options{
		SessionStorage: sessionStorage,
		Logger:         logger,
	})

	// Create auth flow
	flow := tdauth.NewFlow(
		TerminalUserAuthenticator{PhoneNumber: phoneNumber},
		tdauth.SendCodeOptions{},
	)

	slog.Info("starting telegram client connection")
	slog.Info("NOTE: If authentication hangs, check that your system clock is synchronized")
	slog.Info("Telegram will reject connections if your clock is out of sync")

	// Run client with authentication
	return waiter.Run(ctx, func(ctx context.Context) error {
		slog.Info("waiter.Run callback started")
		err := client.Run(ctx, func(ctx context.Context) error {
			slog.Info("client.Run callback started, calling Auth().IfNecessary")
			// Authenticate if necessary
			if err := client.Auth().IfNecessary(ctx, flow); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			slog.Info("Auth().IfNecessary completed")

			// Get user info
			self, err := client.Self(ctx)
			if err != nil {
				return fmt.Errorf("failed to get self info: %w", err)
			}

			name := self.FirstName
			if self.Username != "" {
				name = fmt.Sprintf("%s (@%s)", name, self.Username)
			}
			slog.Info("telegram authenticated", "as", name)

			// Run the user's function
			return runner(ctx, client)
		})
		if err != nil {
			slog.Error("client.Run failed", "error", err)
		}
		return err
	})
}
