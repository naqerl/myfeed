package summary

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"

	"github.com/scipunch/myfeed/config"
)

//go:embed *.prompt
var prompts embed.FS

const (
	agentName  = "summary"
	promptName = "summary"
)

// SummaryAgent uses Gemini to summarize content
type SummaryAgent struct {
	prompt *ai.Prompt
	g      *genkit.Genkit
}

// New creates a new summary agent with its own genkit instance.
// It fails fast if the prompt is not found or Gemini credentials are invalid.
func New(ctx context.Context, creds config.GeminiCredentials) (*SummaryAgent, error) {
	if !creds.IsValid() {
		return nil, fmt.Errorf("invalid Gemini credentials: API key and model must be set")
	}

	// Initialize genkit with Google Generative AI plugin
	g := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{
			APIKey: creds.APIKey,
		}),
		genkit.WithPromptFS(prompts),
		genkit.WithPromptDir("."),
		genkit.WithDefaultModel(fmt.Sprintf("googleai/%s", creds.Model)),
	)

	// Fail fast if prompt wasn't found
	prompt := genkit.LookupPrompt(g, promptName)
	if prompt == nil {
		log.Fatalf("prompt '%s' not found in embedded files", promptName)
	}

	return &SummaryAgent{
		prompt: &prompt,
		g:      g,
	}, nil
}

// Name returns the agent identifier
func (a *SummaryAgent) Name() string {
	return agentName
}

// Process summarizes the provided content using Gemini
func (a *SummaryAgent) Process(ctx context.Context, content string) (string, error) {
	resp, err := (*a.prompt).Execute(ctx,
		ai.WithInput(map[string]any{"content": content}))
	if err != nil {
		return "", fmt.Errorf("failed to execute summary prompt: %w", err)
	}

	return resp.Text(), nil
}
