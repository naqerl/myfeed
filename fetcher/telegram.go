package fetcher

// TelegramFetcher fetches feeds from Telegram channels
type TelegramFetcher struct {
	// Future: Add Telegram API client configuration
}

// NewTelegramFetcher creates a new Telegram fetcher
func NewTelegramFetcher() *TelegramFetcher {
	return &TelegramFetcher{}
}

// Fetch retrieves a feed from a Telegram channel
func (f *TelegramFetcher) Fetch(url string) (Feed, error) {
	panic("TelegramChannel resource type not implemented yet")
}
