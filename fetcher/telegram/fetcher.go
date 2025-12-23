package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"

	"github.com/scipunch/myfeed/fetcher/types"
)

const (
	defaultMessageLimit = 50
)

// TelegramFetcher fetches feeds from Telegram channels
type TelegramFetcher struct {
	configDir   string
	appID       int
	appHash     string
	phoneNumber string
}

// NewTelegramFetcher creates a new Telegram fetcher with provided credentials
func NewTelegramFetcher(configDir string, appID int, appHash string, phoneNumber string) *TelegramFetcher {
	return &TelegramFetcher{
		configDir:   configDir,
		appID:       appID,
		appHash:     appHash,
		phoneNumber: phoneNumber,
	}
}

// Fetch retrieves a feed from a Telegram channel
func (f *TelegramFetcher) Fetch(ctx context.Context, url string) (types.Feed, error) {
	var feed types.Feed

	// Parse URL to extract channel username
	username, err := parseChannelURL(url)
	if err != nil {
		return feed, fmt.Errorf("invalid channel URL: %w", err)
	}

	// Run with authenticated client
	err = RunWithAuth(ctx, f.configDir, f.appID, f.appHash, f.phoneNumber, func(ctx context.Context, client *telegram.Client) error {
		api := client.API()

		// Resolve channel username
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: username,
		})
		if err != nil {
			return fmt.Errorf("failed to resolve channel @%s: %w", username, err)
		}

		// Extract channel from resolved peer
		var channel *tg.Channel
		for _, chat := range resolved.Chats {
			if ch, ok := chat.(*tg.Channel); ok {
				channel = ch
				break
			}
		}

		if channel == nil {
			return fmt.Errorf("channel @%s not found in resolved peers", username)
		}

		// Check if it's actually a channel
		if !channel.Broadcast {
			panic(fmt.Sprintf("@%s is not a channel (it's a group or supergroup)", username))
		}

		// Set feed metadata
		feed.Title = channel.Title
		feed.Description = fmt.Sprintf("Telegram channel @%s", username)

		// Try to get full channel info for description
		fullChan, err := api.ChannelsGetFullChannel(ctx, &tg.InputChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		})
		if err == nil {
			if chatFull, ok := fullChan.FullChat.(*tg.ChannelFull); ok {
				if chatFull.About != "" {
					feed.Description = chatFull.About
				}
			}
		}

		// Fetch channel messages
		inputPeer := &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		}

		messagesData, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:  inputPeer,
			Limit: defaultMessageLimit,
		})
		if err != nil {
			return fmt.Errorf("failed to fetch messages from @%s: %w", username, err)
		}

		// Extract messages
		var messages []tg.MessageClass
		switch m := messagesData.(type) {
		case *tg.MessagesMessages:
			messages = m.Messages
		case *tg.MessagesMessagesSlice:
			messages = m.Messages
		case *tg.MessagesChannelMessages:
			messages = m.Messages
		case *tg.MessagesMessagesNotModified:
			slog.Warn("messages not modified", "channel", username)
			return nil
		default:
			return fmt.Errorf("unexpected messages type: %T", messagesData)
		}

		// Convert messages to feed items
		feed.Items = make([]types.FeedItem, 0, len(messages))
		for _, msgClass := range messages {
			msg, ok := msgClass.(*tg.Message)
			if !ok {
				continue // Skip service messages
			}

			// Skip empty messages
			if msg.Message == "" {
				continue
			}

			item := types.FeedItem{
				Title:       truncateText(msg.Message, 100), // Use first 100 chars as title
				Link:        fmt.Sprintf("https://t.me/%s/%d", username, msg.ID),
				Description: msg.Message,
				Published:   time.Unix(int64(msg.Date), 0),
				GUID:        fmt.Sprintf("%d", msg.ID), // Use message ID as GUID
			}

			feed.Items = append(feed.Items, item)
		}

		// Reverse the items to get oldest first (Telegram API returns newest first)
		for i, j := 0, len(feed.Items)-1; i < j; i, j = i+1, j-1 {
			feed.Items[i], feed.Items[j] = feed.Items[j], feed.Items[i]
		}

		slog.Info("fetched Telegram channel", "channel", username, "messages", len(feed.Items))
		return nil
	})

	return feed, err
}

// parseChannelURL extracts the channel username from various URL formats
// Supports:
//   - https://t.me/channelname
//   - http://t.me/channelname
//   - t.me/channelname
//   - @channelname
//   - channelname
func parseChannelURL(url string) (string, error) {
	url = strings.TrimSpace(url)

	// Remove protocol if present
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove t.me/ if present
	url = strings.TrimPrefix(url, "t.me/")

	// Remove @ if present
	url = strings.TrimPrefix(url, "@")

	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Validate username (basic validation)
	if url == "" {
		return "", fmt.Errorf("empty channel username")
	}

	// Username should not contain slashes (no deep links)
	if strings.Contains(url, "/") {
		return "", fmt.Errorf("invalid channel URL format: %s", url)
	}

	return url, nil
}

// truncateText truncates text to maxLen characters, adding "..." if truncated
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Try to truncate at word boundary
	truncated := text[:maxLen]
	if idx := strings.LastIndex(truncated, " "); idx > maxLen/2 {
		truncated = truncated[:idx]
	}

	return truncated + "..."
}
