package telegram

import (
	"fmt"
	"html"
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
func (p Parser) Parse(item types.FeedItem) (parser.Response, error) {
	// For Telegram messages, the content is in Description field
	// Link field is used as the cache key
	html := convertTelegramToHTML(item.Description)
	return Response{HTML: html}, nil
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
