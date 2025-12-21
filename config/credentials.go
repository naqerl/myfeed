package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const baseCredPath = "myfeed/creds.toml"

// Credentials holds all application credentials
type Credentials struct {
	Telegram TelegramCredentials `toml:"telegram"`
	Gemini   GeminiCredentials   `toml:"gemini"`
}

// TelegramCredentials holds Telegram API credentials
type TelegramCredentials struct {
	AppID       int    `toml:"api_id"`
	AppHash     string `toml:"api_hash"`
	PhoneNumber string `toml:"phone"`
}

// IsValid checks if telegram credentials are fully populated
func (tc TelegramCredentials) IsValid() bool {
	return tc.AppID != 0 && tc.AppHash != "" && tc.PhoneNumber != ""
}

// GeminiCredentials holds Google Gemini API credentials
type GeminiCredentials struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"` // e.g., "gemini-2.0-flash-exp"
}

// IsValid checks if Gemini credentials are fully populated
func (gc GeminiCredentials) IsValid() bool {
	return gc.APIKey != "" && gc.Model != ""
}

// ReadCredentials reads credentials from the specified path
func ReadCredentials(path string) (Credentials, error) {
	var creds Credentials

	data, err := os.ReadFile(path)
	if err != nil {
		return creds, err
	}

	if _, err := toml.Decode(string(data), &creds); err != nil {
		return creds, fmt.Errorf("failed to decode credentials at %s: %w", path, err)
	}

	return creds, nil
}

// WriteCredentials writes credentials to the specified path
func WriteCredentials(path string, creds Credentials) error {
	blob, err := toml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}

	basePath := filepath.Dir(path)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create credentials directory at '%s': %w", basePath, err)
	}

	// Write with restrictive permissions (only owner can read/write)
	if err := os.WriteFile(path, blob, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file at '%s': %w", path, err)
	}

	return nil
}

// DefaultCredentialsPath returns the default path for credentials file
func DefaultCredentialsPath() string {
	var xdgHome = os.Getenv("XDG_CONFIG_HOME")
	if xdgHome != "" {
		return filepath.Join(xdgHome, baseCredPath)
	}

	var home = os.Getenv("HOME")
	if home != "" {
		return filepath.Join(home, ".config", baseCredPath)
	}

	panic("unable to determine credentials file path")
}

// PromptTelegramCredentials prompts the user for Telegram credentials
func PromptTelegramCredentials() (TelegramCredentials, error) {
	var creds TelegramCredentials

	fmt.Println("Telegram credentials not found. Please provide the following information:")
	fmt.Println()
	fmt.Println("To get API_ID and API_HASH:")
	fmt.Println("  1. Go to https://my.telegram.org")
	fmt.Println("  2. Log in with your phone number")
	fmt.Println("  3. Click 'API development tools'")
	fmt.Println("  4. Create a new application")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Prompt for API_ID
	fmt.Print("Enter API_ID: ")
	appIDStr, err := reader.ReadString('\n')
	if err != nil {
		return creds, fmt.Errorf("failed to read API_ID: %w", err)
	}
	appIDStr = strings.TrimSpace(appIDStr)
	if _, err := fmt.Sscanf(appIDStr, "%d", &creds.AppID); err != nil {
		return creds, fmt.Errorf("invalid API_ID format: %w", err)
	}

	// Prompt for API_HASH
	fmt.Print("Enter API_HASH: ")
	appHash, err := reader.ReadString('\n')
	if err != nil {
		return creds, fmt.Errorf("failed to read API_HASH: %w", err)
	}
	creds.AppHash = strings.TrimSpace(appHash)

	// Prompt for phone number
	fmt.Print("Enter phone number in international format (e.g. +1234567890): ")
	phone, err := reader.ReadString('\n')
	if err != nil {
		return creds, fmt.Errorf("failed to read phone number: %w", err)
	}
	creds.PhoneNumber = strings.TrimSpace(phone)

	// Validate credentials
	if !creds.IsValid() {
		return creds, fmt.Errorf("all credential fields are required")
	}

	fmt.Println()
	fmt.Println("Next, you will need to authenticate with Telegram.")
	fmt.Println("A verification code will be sent to your phone...")
	fmt.Println()

	return creds, nil
}

// LoadOrPromptTelegramCredentials loads telegram credentials or prompts for them
func LoadOrPromptTelegramCredentials(credPath string) (TelegramCredentials, error) {
	// Try to load existing credentials
	creds, err := ReadCredentials(credPath)
	if err == nil && creds.Telegram.IsValid() {
		return creds.Telegram, nil
	}

	// Credentials not found or invalid, prompt user
	telegramCreds, err := PromptTelegramCredentials()
	if err != nil {
		return TelegramCredentials{}, err
	}

	// Save credentials
	creds.Telegram = telegramCreds
	if err := WriteCredentials(credPath, creds); err != nil {
		return telegramCreds, fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("Credentials saved to %s\n", credPath)
	fmt.Println()

	return telegramCreds, nil
}
