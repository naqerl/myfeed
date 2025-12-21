package agent

import (
	"context"
	"fmt"

	"github.com/scipunch/myfeed/agent/summary"
	"github.com/scipunch/myfeed/config"
)

// InitAgents creates agents based on the requested agent types.
// It fails fast if any agent initialization fails (e.g., missing credentials, invalid prompts).
// Returns a map of agent name -> agent instance.
// All agents are automatically wrapped with retry logic (exponential backoff, 5-minute timeout).
func InitAgents(ctx context.Context, agentTypes []string, creds config.GeminiCredentials) (map[string]Agent, error) {
	agents := make(map[string]Agent)
	retryConfig := DefaultRetryConfig()

	for _, agentType := range agentTypes {
		var baseAgent Agent
		var err error

		switch agentType {
		case "summary":
			baseAgent, err = summary.New(ctx, creds)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize summary agent: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown agent type: %s", agentType)
		}

		// Wrap with retry logic
		agents[agentType] = WithRetry(baseAgent, retryConfig)
	}

	return agents, nil
}

// CollectUniqueAgentTypes extracts unique agent types from resource configurations
func CollectUniqueAgentTypes(resources []config.ResourceConfig) []string {
	typeSet := make(map[string]bool)
	for _, resource := range resources {
		for _, agentType := range resource.Agents {
			typeSet[agentType] = true
		}
	}

	var types []string
	for agentType := range typeSet {
		types = append(types, agentType)
	}
	return types
}
