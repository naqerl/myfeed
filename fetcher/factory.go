package fetcher

import (
	"fmt"

	"github.com/scipunch/myfeed/config"
)

// GetFetchers creates a map of resource types to their corresponding fetchers
func GetFetchers(resourceTypes []config.ResourceType) (map[config.ResourceType]FeedFetcher, error) {
	fetchers := make(map[config.ResourceType]FeedFetcher)

	for _, rt := range resourceTypes {
		// Skip if we already have a fetcher for this type
		if fetchers[rt] != nil {
			continue
		}

		switch rt {
		case config.RSS:
			fetchers[rt] = NewRSSFetcher()
		case config.TelegramChannel:
			fetchers[rt] = NewTelegramFetcher()
		default:
			return nil, fmt.Errorf("unknown resource type: %s", rt)
		}
	}

	return fetchers, nil
}
