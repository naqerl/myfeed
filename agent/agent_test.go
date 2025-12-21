package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockAgent is a test agent that can be configured to fail
type mockAgent struct {
	name         string
	failCount    int // Number of times to fail before succeeding
	currentFails int
	processDelay time.Duration
}

func (m *mockAgent) Name() string {
	return m.name
}

func (m *mockAgent) Process(ctx context.Context, content string) (string, error) {
	if m.processDelay > 0 {
		time.Sleep(m.processDelay)
	}

	if m.currentFails < m.failCount {
		m.currentFails++
		return "", errors.New("Error 429, Message: You exceeded your current quota, Status: RESOURCE_EXHAUSTED, retryDelay:2s")
	}

	return "processed: " + content, nil
}

func TestWithRetry_Success(t *testing.T) {
	mock := &mockAgent{name: "test", failCount: 0}
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Timeout:        5 * time.Second,
	}

	agent := WithRetry(mock, config)

	result, err := agent.Process(context.Background(), "test content")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if result != "processed: test content" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestWithRetry_SuccessAfterRetries(t *testing.T) {
	mock := &mockAgent{name: "test", failCount: 2}
	config := RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Timeout:        5 * time.Second,
	}

	agent := WithRetry(mock, config)

	start := time.Now()
	result, err := agent.Process(context.Background(), "test content")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}

	if result != "processed: test content" {
		t.Errorf("unexpected result: %s", result)
	}

	// Should have waited at least for the backoffs
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected some delay due to retries, got %v", elapsed)
	}

	if mock.currentFails != 2 {
		t.Errorf("expected 2 failures, got %d", mock.currentFails)
	}
}

func TestWithRetry_ExceedsMaxRetries(t *testing.T) {
	mock := &mockAgent{name: "test", failCount: 10}
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
		Timeout:        5 * time.Second,
	}

	agent := WithRetry(mock, config)

	_, err := agent.Process(context.Background(), "test content")
	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}

	if !strings.Contains(err.Error(), "max retries") {
		t.Errorf("expected max retries error, got: %v", err)
	}

	// Should have failed 4 times (initial + 3 retries)
	if mock.currentFails != 4 {
		t.Errorf("expected 4 failures, got %d", mock.currentFails)
	}
}

func TestWithRetry_Timeout(t *testing.T) {
	mock := &mockAgent{
		name:         "test",
		failCount:    100,
		processDelay: 200 * time.Millisecond,
	}
	config := RetryConfig{
		MaxRetries:     20,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
		Timeout:        500 * time.Millisecond, // Short timeout
	}

	agent := WithRetry(mock, config)

	start := time.Now()
	_, err := agent.Process(context.Background(), "test content")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected timeout/cancelled error, got: %v", err)
	}

	// Should have timed out around 500ms, not waited for all retries
	if elapsed > 1*time.Second {
		t.Errorf("took too long (%v), should have timed out quickly", elapsed)
	}
}

func TestWithRetry_ContextCancellation(t *testing.T) {
	mock := &mockAgent{name: "test", failCount: 100}
	config := RetryConfig{
		MaxRetries:     10,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
		Timeout:        5 * time.Second,
	}

	agent := WithRetry(mock, config)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := agent.Process(ctx, "test content")
	if err == nil {
		t.Fatal("expected error after context cancellation, got nil")
	}

	if !strings.Contains(err.Error(), "cancel") && !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected cancellation error, got: %v", err)
	}
}

// mockNonRetryableAgent returns non-retryable errors
type mockNonRetryableAgent struct {
	name string
}

func (m *mockNonRetryableAgent) Name() string {
	return m.name
}

func (m *mockNonRetryableAgent) Process(ctx context.Context, content string) (string, error) {
	return "", errors.New("invalid input: malformed content")
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	mock := &mockNonRetryableAgent{name: "test"}
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Timeout:        5 * time.Second,
	}

	agent := WithRetry(mock, config)

	_, err := agent.Process(context.Background(), "test content")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "non-retryable") {
		t.Errorf("expected non-retryable error, got: %v", err)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		err       error
		retryable bool
	}{
		{nil, false},
		{errors.New("RESOURCE_EXHAUSTED"), true},
		{errors.New("quota exceeded"), true},
		{errors.New("Error 429"), true},
		{errors.New("Error 503"), true},
		{errors.New("rate limit exceeded"), true},
		{errors.New("invalid input"), false},
		{errors.New("authentication failed"), false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.err), func(t *testing.T) {
			got := isRetryable(tt.err)
			if got != tt.retryable {
				t.Errorf("isRetryable(%v) = %v, want %v", tt.err, got, tt.retryable)
			}
		})
	}
}

func TestExtractRetryDelay(t *testing.T) {
	tests := []struct {
		err      error
		expected time.Duration
	}{
		{nil, 0},
		{errors.New("Please retry in 12.5s"), 12500 * time.Millisecond},
		{errors.New("retryDelay:10s"), 10 * time.Second},
		{errors.New("no delay info"), 0},
		{errors.New("retry in 1.5s, then check status"), 1500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.err), func(t *testing.T) {
			got := extractRetryDelay(tt.err)
			if got != tt.expected {
				t.Errorf("extractRetryDelay(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries <= 0 {
		t.Error("MaxRetries should be positive")
	}

	if config.InitialBackoff <= 0 {
		t.Error("InitialBackoff should be positive")
	}

	if config.MaxBackoff <= config.InitialBackoff {
		t.Error("MaxBackoff should be greater than InitialBackoff")
	}

	if config.Timeout != 5*time.Minute {
		t.Errorf("Timeout should be 5 minutes, got %v", config.Timeout)
	}
}
