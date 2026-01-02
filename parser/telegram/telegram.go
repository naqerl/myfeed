package telegram

import (
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/scipunch/myfeed/fetcher/types"
	"github.com/scipunch/myfeed/parser"
)

// Parser parses Telegram messages and converts them to HTML
type Parser struct{}

// New creates a new Telegram parser
func New() (Parser, error) {
	return Parser{}, nil
}

// Response represents a parsed Telegram message
type Response struct {
	HTML string
}

func (r Response) String() string {
	return r.HTML
}

// Parse takes a FeedItem and converts the Description (Telegram message content) to HTML
// Uses item.Link as the cache key, but processes item.Description as the content
// Also includes any media attachments (photos) in the HTML
func (p Parser) Parse(item types.FeedItem) (parser.Response, error) {
	var htmlBuilder strings.Builder

	// Add media (photos) before the text content
	for _, media := range item.Media {
		if media.Type == "photo" {
			if media.LocalPath != "" {
				// Image with successful download
				// Use relative path: media/filename
				filename := filepath.Base(media.LocalPath)
				relativePath := filepath.Join("media", filename)

				htmlBuilder.WriteString(fmt.Sprintf(
					`<img src="%s" alt="%s" style="max-width: 100%%; height: auto; margin-bottom: 1em;" width="%d" height="%d">`,
					relativePath,
					escapeHTML(media.Caption),
					media.Width,
					media.Height,
				))
				htmlBuilder.WriteString("\n")
			} else if media.Caption != "" {
				// Failed download - show error message
				htmlBuilder.WriteString(fmt.Sprintf(
					`<div style="padding: 1em; background: #fee; border: 1em solid #fcc; margin-bottom: 1em;">%s</div>`,
					escapeHTML(media.Caption),
				))
				htmlBuilder.WriteString("\n")
			}
		}
	}

	// Add text content
	if item.Description != "" {
		textHTML := convertTelegramToHTML(item.Description)
		htmlBuilder.WriteString(textHTML)
	}

	return Response{HTML: htmlBuilder.String()}, nil
}

// escapeHTML escapes HTML special characters
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// ParseMessage converts a Telegram message text to HTML
// This is the actual useful method for Telegram messages
func (p Parser) ParseMessage(message string) Response {
	html := convertTelegramToHTML(message)
	return Response{HTML: html}
}

// convertTelegramToHTML converts Telegram formatting to HTML
// Telegram supports:
// - **bold**
// - __italic__
// - `code`
// - ```pre```
// - [text](url) - links
func convertTelegramToHTML(text string) string {
	if text == "" {
		return ""
	}

	// Escape HTML first
	text = html.EscapeString(text)

	// Convert code blocks (```code```)
	codeBlockRe := regexp.MustCompile("```([^`]+)```")
	text = codeBlockRe.ReplaceAllString(text, "<pre><code>$1</code></pre>")

	// Convert inline code (`code`)
	inlineCodeRe := regexp.MustCompile("`([^`]+)`")
	text = inlineCodeRe.ReplaceAllString(text, "<code>$1</code>")

	// Convert bold (**text**)
	boldRe := regexp.MustCompile(`\*\*([^\*]+)\*\*`)
	text = boldRe.ReplaceAllString(text, "<strong>$1</strong>")

	// Convert italic (__text__)
	italicRe := regexp.MustCompile(`__([^_]+)__`)
	text = italicRe.ReplaceAllString(text, "<em>$1</em>")

	// Convert strikethrough (~~text~~)
	strikeRe := regexp.MustCompile(`~~([^~]+)~~`)
	text = strikeRe.ReplaceAllString(text, "<del>$1</del>")

	// Convert links [text](url)
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)
	text = linkRe.ReplaceAllString(text, `<a href="$2">$1</a>`)

	// Convert newlines to <br>
	text = strings.ReplaceAll(text, "\n", "<br>\n")

	// Wrap in paragraph
	return fmt.Sprintf("<p>%s</p>", text)
}
