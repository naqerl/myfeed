package cache

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Cache provides caching for parser and agent outputs using sqlc-generated queries
type Cache struct {
	db      *sql.DB
	queries *Queries
}

// CacheStats contains cache statistics
type CacheStats struct {
	ParserEntries int
	AgentEntries  int
	OldestEntry   time.Time
}

// NewCache initializes cache database at the given path
func NewCache(dbPath string) (*Cache, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	// Execute schema
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize cache schema: %w", err)
	}

	return &Cache{
		db:      db,
		queries: New(db),
	}, nil
}

// GetParserOutput retrieves cached parser output
// Returns: (output, found, error)
func (c *Cache) GetParserOutput(url, parserType string) ([]byte, bool, error) {
	ctx := context.Background()

	output, err := c.queries.GetParserOutput(ctx, GetParserOutputParams{
		Url:        url,
		ParserType: parserType,
	})

	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		slog.Warn("parser cache read error", "error", err, "url", truncate(url, 50))
		return nil, false, nil // Treat errors as cache miss
	}

	// Update accessed_at
	accessedAt := time.Now().Unix()
	_ = c.queries.UpdateParserAccessTime(ctx, UpdateParserAccessTimeParams{
		AccessedAt: accessedAt,
		Url:        url,
		ParserType: parserType,
	})

	return []byte(output), true, nil
}

// SetParserOutput stores parser output in cache
func (c *Cache) SetParserOutput(url, parserType string, output []byte) error {
	ctx := context.Background()
	now := time.Now().Unix()

	err := c.queries.SetParserOutput(ctx, SetParserOutputParams{
		Url:        url,
		ParserType: parserType,
		OutputData: string(output),
		CreatedAt:  now,
		AccessedAt: now,
	})

	if err != nil {
		slog.Warn("parser cache write error", "error", err, "url", truncate(url, 50))
		return err
	}

	return nil
}

// GetAgentOutput retrieves cached agent output
// agentPipeline should be slice of agent names (e.g., ["summary", "translate"])
func (c *Cache) GetAgentOutput(url, parserType string, agentPipeline []string) (string, bool, error) {
	ctx := context.Background()
	pipeline := strings.Join(agentPipeline, ",")

	output, err := c.queries.GetAgentOutput(ctx, GetAgentOutputParams{
		Url:           url,
		ParserType:    parserType,
		AgentPipeline: pipeline,
	})

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		slog.Warn("agent cache read error", "error", err, "url", truncate(url, 50))
		return "", false, nil
	}

	// Update accessed_at
	accessedAt := time.Now().Unix()
	_ = c.queries.UpdateAgentAccessTime(ctx, UpdateAgentAccessTimeParams{
		AccessedAt:    accessedAt,
		Url:           url,
		ParserType:    parserType,
		AgentPipeline: pipeline,
	})

	return output, true, nil
}

// SetAgentOutput stores agent output in cache
func (c *Cache) SetAgentOutput(url, parserType string, agentPipeline []string, output string) error {
	ctx := context.Background()
	now := time.Now().Unix()
	pipeline := strings.Join(agentPipeline, ",")

	err := c.queries.SetAgentOutput(ctx, SetAgentOutputParams{
		Url:           url,
		ParserType:    parserType,
		AgentPipeline: pipeline,
		OutputData:    output,
		CreatedAt:     now,
		AccessedAt:    now,
	})

	if err != nil {
		slog.Warn("agent cache write error", "error", err, "url", truncate(url, 50))
		return err
	}

	return nil
}

// Clear removes all cache entries
func (c *Cache) Clear() error {
	ctx := context.Background()

	if err := c.queries.DeleteParserCache(ctx); err != nil {
		return fmt.Errorf("failed to clear parser cache: %w", err)
	}
	if err := c.queries.DeleteAgentCache(ctx); err != nil {
		return fmt.Errorf("failed to clear agent cache: %w", err)
	}
	return nil
}

// Stats returns cache statistics
func (c *Cache) Stats() (CacheStats, error) {
	ctx := context.Background()
	var stats CacheStats

	parserCount, err := c.queries.CountParserEntries(ctx)
	if err != nil {
		return stats, err
	}
	stats.ParserEntries = int(parserCount)

	agentCount, err := c.queries.CountAgentEntries(ctx)
	if err != nil {
		return stats, err
	}
	stats.AgentEntries = int(agentCount)

	oldest, err := c.queries.GetOldestCacheEntry(ctx)
	if err != nil && err != sql.ErrNoRows {
		return stats, err
	}

	// Handle the interface{} type from sqlc
	if oldest != nil {
		switch v := oldest.(type) {
		case int64:
			if v > 0 {
				stats.OldestEntry = time.Unix(v, 0)
			}
		case float64:
			if v > 0 {
				stats.OldestEntry = time.Unix(int64(v), 0)
			}
		}
	}

	return stats, nil
}

// Close closes the cache database
func (c *Cache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// DefaultCachePath returns the default cache database path
func DefaultCachePath() string {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "cache.db" // Fallback to current directory
		}
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "myfeed", "cache.db")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
