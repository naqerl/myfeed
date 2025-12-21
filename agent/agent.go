package agent

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"
)

// Agent defines the interface for content processing agents.
// Agents can perform various transformations on content such as
// summarization, translation, formatting, or content generation.
type Agent interface {
	// Process takes content and returns processed markdown
	Process(ctx context.Context, content string) (string, error)

	// Name returns the agent identifier (e.g., "summary")
	Name() string
}

// RetryConfig defines retry behavior for agent operations
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retry attempts
	InitialBackoff time.Duration // Initial backoff duration
	MaxBackoff     time.Duration // Maximum backoff duration
	Timeout        time.Duration // Overall timeout for the operation
}

// DefaultRetryConfig returns sensible defaults for API retries
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     10,               // Allow up to 10 retries
		InitialBackoff: 1 * time.Second,  // Start with 1 second
		MaxBackoff:     30 * time.Second, // Cap at 30 seconds
		Timeout:        5 * time.Minute,  // Total timeout of 5 minutes
	}
}

// WithRetry wraps an agent with retry logic using exponential backoff
func WithRetry(agent Agent, config RetryConfig) Agent {
	return &retryAgent{
		underlying: agent,
		config:     config,
	}
}

type retryAgent struct {
	underlying Agent
	config     RetryConfig
}

func (r *retryAgent) Name() string {
	return r.underlying.Name()
}

func (r *retryAgent) Process(ctx context.Context, content string) (string, error) {
	// Create a context with overall timeout
	ctx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	var lastErr error
	backoff := r.config.InitialBackoff

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check if context is already cancelled
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("operation timed out after %d attempts: %w", attempt, ctx.Err())
		default:
		}

		// Try processing
		result, err := r.underlying.Process(ctx, content)
		if err == nil {
			if attempt > 0 {
				slog.Info("agent succeeded after retries",
					"agent", r.Name(),
					"attempts", attempt+1)
			}
			return result, nil
		}

		lastErr = err

		// Check if error is retryable (quota/rate limit errors)
		if !isRetryable(err) {
			return "", fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't sleep after the last attempt
		if attempt == r.config.MaxRetries {
			break
		}

		// Calculate backoff with exponential growth
		sleepDuration := time.Duration(float64(backoff) * math.Pow(2, float64(attempt)))
		if sleepDuration > r.config.MaxBackoff {
			sleepDuration = r.config.MaxBackoff
		}

		// Check if suggested retry delay from API is available
		if suggestedDelay := extractRetryDelay(err); suggestedDelay > 0 {
			sleepDuration = suggestedDelay
			if sleepDuration > r.config.MaxBackoff {
				sleepDuration = r.config.MaxBackoff
			}
		}

		slog.Warn("agent call failed, retrying",
			"agent", r.Name(),
			"attempt", attempt+1,
			"max_attempts", r.config.MaxRetries+1,
			"retry_in", sleepDuration,
			"error", err)

		// Wait before retry, respecting context cancellation
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("operation cancelled during backoff: %w", ctx.Err())
		case <-time.After(sleepDuration):
			// Continue to next attempt
		}
	}

	return "", fmt.Errorf("max retries (%d) exceeded: %w", r.config.MaxRetries, lastErr)
}

// isRetryable determines if an error should trigger a retry
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Quota and rate limit errors are retryable
	retryablePatterns := []string{
		"RESOURCE_EXHAUSTED",
		"quota exceeded",
		"rate limit",
		"429",
		"503", // Service unavailable
		"500", // Internal server error (sometimes transient)
	}

	errLower := strings.ToLower(errStr)
	for _, pattern := range retryablePatterns {
		if strings.Contains(errLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// extractRetryDelay attempts to extract suggested retry delay from error message
func extractRetryDelay(err error) time.Duration {
	if err == nil {
		return 0
	}

	errStr := err.Error()

	// Look for "retry in X.Xs" or "retryDelay:Xs" patterns
	if idx := strings.Index(errStr, "retry in "); idx != -1 {
		// Extract the duration string (e.g., "12.129100495s")
		start := idx + len("retry in ")
		end := start
		for end < len(errStr) && (errStr[end] >= '0' && errStr[end] <= '9' || errStr[end] == '.' || errStr[end] == 's') {
			end++
		}
		if end > start {
			durationStr := errStr[start:end]
			if d, err := time.ParseDuration(durationStr); err == nil {
				return d
			}
		}
	}

	// Look for retryDelay:12s pattern
	if idx := strings.Index(errStr, "retryDelay:"); idx != -1 {
		start := idx + len("retryDelay:")
		end := start
		for end < len(errStr) && (errStr[end] >= '0' && errStr[end] <= '9' || errStr[end] == 's') {
			end++
		}
		if end > start {
			durationStr := errStr[start:end]
			if d, err := time.ParseDuration(durationStr); err == nil {
				return d
			}
		}
	}

	return 0
}
