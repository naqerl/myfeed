package cache

import (
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

// Cache provides caching for parser and agent outputs
type Cache struct {
	db *sql.DB
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

	return &Cache{db: db}, nil
}

// GetParserOutput retrieves cached parser output
// Returns: (output, found, error)
func (c *Cache) GetParserOutput(url, parserType string) ([]byte, bool, error) {
	var output []byte
	accessedAt := time.Now().Unix()

	err := c.db.QueryRow(
		"SELECT output_data FROM parser_cache WHERE url = ? AND parser_type = ?",
		url, parserType,
	).Scan(&output)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		slog.Warn("parser cache read error", "error", err, "url", truncate(url, 50))
		return nil, false, nil // Treat errors as cache miss
	}

	// Update accessed_at
	_, _ = c.db.Exec(
		"UPDATE parser_cache SET accessed_at = ? WHERE url = ? AND parser_type = ?",
		accessedAt, url, parserType,
	)

	return output, true, nil
}

// SetParserOutput stores parser output in cache
func (c *Cache) SetParserOutput(url, parserType string, output []byte) error {
	now := time.Now().Unix()

	_, err := c.db.Exec(`
		INSERT OR REPLACE INTO parser_cache 
		(url, parser_type, output_data, created_at, accessed_at)
		VALUES (?, ?, ?, ?, ?)
	`, url, parserType, output, now, now)

	if err != nil {
		slog.Warn("parser cache write error", "error", err, "url", truncate(url, 50))
		return err
	}

	return nil
}

// GetAgentOutput retrieves cached agent output
// agentPipeline should be slice of agent names (e.g., ["summary", "translate"])
func (c *Cache) GetAgentOutput(url, parserType string, agentPipeline []string) (string, bool, error) {
	pipeline := strings.Join(agentPipeline, ",")

	var output string
	accessedAt := time.Now().Unix()

	err := c.db.QueryRow(
		"SELECT output_data FROM agent_cache WHERE url = ? AND parser_type = ? AND agent_pipeline = ?",
		url, parserType, pipeline,
	).Scan(&output)

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		slog.Warn("agent cache read error", "error", err, "url", truncate(url, 50))
		return "", false, nil
	}

	// Update accessed_at
	_, _ = c.db.Exec(
		"UPDATE agent_cache SET accessed_at = ? WHERE url = ? AND parser_type = ? AND agent_pipeline = ?",
		accessedAt, url, parserType, pipeline,
	)

	return output, true, nil
}

// SetAgentOutput stores agent output in cache
func (c *Cache) SetAgentOutput(url, parserType string, agentPipeline []string, output string) error {
	now := time.Now().Unix()
	pipeline := strings.Join(agentPipeline, ",")

	_, err := c.db.Exec(`
		INSERT OR REPLACE INTO agent_cache
		(url, parser_type, agent_pipeline, output_data, created_at, accessed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, url, parserType, pipeline, output, now, now)

	if err != nil {
		slog.Warn("agent cache write error", "error", err, "url", truncate(url, 50))
		return err
	}

	return nil
}

// Clear removes all cache entries
func (c *Cache) Clear() error {
	if _, err := c.db.Exec("DELETE FROM parser_cache"); err != nil {
		return fmt.Errorf("failed to clear parser cache: %w", err)
	}
	if _, err := c.db.Exec("DELETE FROM agent_cache"); err != nil {
		return fmt.Errorf("failed to clear agent cache: %w", err)
	}
	return nil
}

// Stats returns cache statistics
func (c *Cache) Stats() (CacheStats, error) {
	var stats CacheStats

	err := c.db.QueryRow("SELECT COUNT(*) FROM parser_cache").Scan(&stats.ParserEntries)
	if err != nil {
		return stats, err
	}

	err = c.db.QueryRow("SELECT COUNT(*) FROM agent_cache").Scan(&stats.AgentEntries)
	if err != nil {
		return stats, err
	}

	var oldestUnix sql.NullInt64
	err = c.db.QueryRow(`
		SELECT MIN(created_at) FROM (
			SELECT created_at FROM parser_cache
			UNION ALL
			SELECT created_at FROM agent_cache
		)
	`).Scan(&oldestUnix)
	if err != nil && err != sql.ErrNoRows {
		return stats, err
	}
	if oldestUnix.Valid && oldestUnix.Int64 > 0 {
		stats.OldestEntry = time.Unix(oldestUnix.Int64, 0)
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
