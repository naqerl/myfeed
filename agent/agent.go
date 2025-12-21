package agent

import "context"

// Agent defines the interface for content processing agents.
// Agents can perform various transformations on content such as
// summarization, translation, formatting, or content generation.
type Agent interface {
	// Process takes content and returns processed markdown
	Process(ctx context.Context, content string) (string, error)

	// Name returns the agent identifier (e.g., "summary")
	Name() string
}
