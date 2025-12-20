package fetcher

import "time"

// Feed represents a collection of items from a feed source
type Feed struct {
	Title       string
	Description string
	Items       []FeedItem
}

// FeedItem represents a single item in a feed
type FeedItem struct {
	Title       string
	Link        string
	Description string
	Published   time.Time
}

// FeedFetcher is an interface for fetching feeds from different sources
type FeedFetcher interface {
	Fetch(url string) (Feed, error)
}
