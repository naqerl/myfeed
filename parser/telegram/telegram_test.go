package telegram

import (
	"strings"
	"testing"
)

func TestConvertTelegramToHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello world",
			expected: "<p>Hello world</p>",
		},
		{
			name:     "bold text",
			input:    "This is **bold** text",
			expected: "<p>This is <strong>bold</strong> text</p>",
		},
		{
			name:     "italic text",
			input:    "This is __italic__ text",
			expected: "<p>This is <em>italic</em> text</p>",
		},
		{
			name:     "inline code",
			input:    "Use `code` for inline",
			expected: "<p>Use <code>code</code> for inline</p>",
		},
		{
			name:     "code block",
			input:    "Here is code:\n```print('hello')```",
			expected: "<p>Here is code:<br>\n<pre><code>print(&#39;hello&#39;)</code></pre></p>",
		},
		{
			name:     "link",
			input:    "Visit [Google](https://google.com)",
			expected: `<p>Visit <a href="https://google.com">Google</a></p>`,
		},
		{
			name:     "strikethrough",
			input:    "This is ~~deleted~~ text",
			expected: "<p>This is <del>deleted</del> text</p>",
		},
		{
			name:     "newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "<p>Line 1<br>\nLine 2<br>\nLine 3</p>",
		},
		{
			name:     "mixed formatting",
			input:    "**Bold** and __italic__ with `code` and [link](https://example.com)",
			expected: `<p><strong>Bold</strong> and <em>italic</em> with <code>code</code> and <a href="https://example.com">link</a></p>`,
		},
		{
			name:     "HTML escaping",
			input:    "This has <script>alert('xss')</script> tags",
			expected: "<p>This has &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt; tags</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTelegramToHTML(tt.input)
			if result != tt.expected {
				t.Errorf("convertTelegramToHTML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseMessage(t *testing.T) {
	parser := Parser{}

	message := "**Important:** This is a test message with __formatting__"
	response := parser.ParseMessage(message)

	result := response.String()
	if !strings.Contains(result, "<strong>Important:</strong>") {
		t.Errorf("Expected bold formatting, got: %s", result)
	}
	if !strings.Contains(result, "<em>formatting</em>") {
		t.Errorf("Expected italic formatting, got: %s", result)
	}
}

func TestParse(t *testing.T) {
	parser, err := New()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	message := "Test message with **bold**"
	response, err := parser.Parse(message)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := response.String()
	if !strings.Contains(result, "<strong>bold</strong>") {
		t.Errorf("Expected formatted output, got: %s", result)
	}
}
