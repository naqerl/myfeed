package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/scipunch/myfeed/fetcher/types"
)

// RSSFetcher fetches RSS feeds using gofeed
type RSSFetcher struct {
	parser *gofeed.Parser
}

// NewRSSFetcher creates a new RSS fetcher
func NewRSSFetcher() *RSSFetcher {
	return &RSSFetcher{
		parser: gofeed.NewParser(),
	}
}

// Fetch retrieves and parses an RSS feed from the given URL
func (f *RSSFetcher) Fetch(ctx context.Context, url string) (types.Feed, error) {
	var feed types.Feed

	gofeedFeed, err := f.parser.ParseURLWithContext(url, ctx)
	if err != nil {
		return feed, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	// Convert gofeed.Feed to our custom Feed type
	feed.Title = gofeedFeed.Title
	feed.Description = gofeedFeed.Description
	feed.Items = make([]types.FeedItem, 0, len(gofeedFeed.Items))

	for _, item := range gofeedFeed.Items {
		feedItem := types.FeedItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			GUID:        item.GUID,
		}

		// Parse published date if available
		if item.PublishedParsed != nil {
			feedItem.Published = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			feedItem.Published = *item.UpdatedParsed
		} else {
			feedItem.Published = time.Time{}
		}

		feed.Items = append(feed.Items, feedItem)
	}

	return feed, nil
}
