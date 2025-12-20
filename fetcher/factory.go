package fetcher

import (
	"fmt"

	"github.com/scipunch/myfeed/config"
	"github.com/scipunch/myfeed/fetcher/telegram"
	"github.com/scipunch/myfeed/fetcher/types"
)

// GetFetchers creates a map of resource types to their corresponding fetchers
func GetFetchers(resourceTypes []config.ResourceType, configDir string) (map[config.ResourceType]types.FeedFetcher, error) {
	fetchers := make(map[config.ResourceType]types.FeedFetcher)

	// Check if telegram is needed
	needsTelegram := false
	for _, rt := range resourceTypes {
		if rt == config.TelegramChannel {
			needsTelegram = true
			break
		}
	}

	// Load or prompt for telegram credentials if needed
	var telegramCreds config.TelegramCredentials
	if needsTelegram {
		credPath := config.DefaultCredentialsPath()
		var err error
		telegramCreds, err = config.LoadOrPromptTelegramCredentials(credPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get telegram credentials: %w", err)
		}
	}

	for _, rt := range resourceTypes {
		// Skip if we already have a fetcher for this type
		if fetchers[rt] != nil {
			continue
		}

		switch rt {
		case config.RSS:
			fetchers[rt] = NewRSSFetcher()
		case config.TelegramChannel:
			fetchers[rt] = telegram.NewTelegramFetcher(configDir, telegramCreds.AppID, telegramCreds.AppHash, telegramCreds.PhoneNumber)
		default:
			return nil, fmt.Errorf("unknown resource type: %s", rt)
		}
	}

	return fetchers, nil
}
