package telegram

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Credentials holds Telegram API credentials
type Credentials struct {
	AppID       int    `json:"app_id"`
	AppHash     string `json:"app_hash"`
	PhoneNumber string `json:"phone_number"`
}

// LoadOrPromptCredentials loads credentials from file or prompts for them
func LoadOrPromptCredentials(configDir string) (Credentials, error) {
	var creds Credentials

	credPath := filepath.Join(configDir, "telegram-credentials.json")

	// Try to load existing credentials
	if data, err := os.ReadFile(credPath); err == nil {
		if err := json.Unmarshal(data, &creds); err == nil {
			// Validate loaded credentials
			if creds.AppID != 0 && creds.AppHash != "" && creds.PhoneNumber != "" {
				return creds, nil
			}
		}
	}

	// Credentials not found or invalid, prompt user
	fmt.Println("Telegram credentials not found. Please provide the following information:")
	fmt.Println()
	fmt.Println("To get APP_ID and APP_HASH:")
	fmt.Println("  1. Go to https://my.telegram.org")
	fmt.Println("  2. Log in with your phone number")
	fmt.Println("  3. Click 'API development tools'")
	fmt.Println("  4. Create a new application")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Prompt for APP_ID
	fmt.Print("Enter APP_ID: ")
	appIDStr, err := reader.ReadString('\n')
	if err != nil {
		return creds, fmt.Errorf("failed to read APP_ID: %w", err)
	}
	appIDStr = strings.TrimSpace(appIDStr)
	if _, err := fmt.Sscanf(appIDStr, "%d", &creds.AppID); err != nil {
		return creds, fmt.Errorf("invalid APP_ID format: %w", err)
	}

	// Prompt for APP_HASH
	fmt.Print("Enter APP_HASH: ")
	appHash, err := reader.ReadString('\n')
	if err != nil {
		return creds, fmt.Errorf("failed to read APP_HASH: %w", err)
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
	if creds.AppID == 0 || creds.AppHash == "" || creds.PhoneNumber == "" {
		return creds, fmt.Errorf("all credential fields are required")
	}

	// Save credentials to file
	if err := SaveCredentials(configDir, creds); err != nil {
		return creds, fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println()
	fmt.Printf("Credentials saved to %s\n", credPath)
	fmt.Println()
	fmt.Println("Next, you will need to authenticate with Telegram.")
	fmt.Println("A verification code will be sent to your phone...")
	fmt.Println()

	return creds, nil
}

// SaveCredentials saves credentials to a JSON file
func SaveCredentials(configDir string, creds Credentials) error {
	credPath := filepath.Join(configDir, "telegram-credentials.json")

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal credentials to JSON
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write to file
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}
