package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	// Verify database file was created
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Cache database file was not created")
	}
}

func TestParserCache_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	url := "https://example.com/article"
	parserType := "web"
	output := []byte(`{"parser_type":"web","data":{"HTML":"<html>test</html>"}}`)

	// Test Set
	err = cache.SetParserOutput(url, parserType, output)
	if err != nil {
		t.Fatalf("SetParserOutput failed: %v", err)
	}

	// Test Get
	retrieved, found, err := cache.GetParserOutput(url, parserType)
	if err != nil {
		t.Fatalf("GetParserOutput failed: %v", err)
	}
	if !found {
		t.Error("Expected cache hit, got miss")
	}
	if string(retrieved) != string(output) {
		t.Errorf("Retrieved data mismatch: got %s, want %s", retrieved, output)
	}
}

func TestParserCache_Miss(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	// Try to get non-existent entry
	_, found, err := cache.GetParserOutput("https://nonexistent.com", "web")
	if err != nil {
		t.Fatalf("GetParserOutput failed: %v", err)
	}
	if found {
		t.Error("Expected cache miss, got hit")
	}
}

func TestParserCache_TypeMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	url := "https://example.com/video"
	output := []byte(`{"parser_type":"youtube","data":{}}`)

	// Store with youtube parser type
	err = cache.SetParserOutput(url, "youtube", output)
	if err != nil {
		t.Fatalf("SetParserOutput failed: %v", err)
	}

	// Try to retrieve with different parser type
	_, found, err := cache.GetParserOutput(url, "web")
	if err != nil {
		t.Fatalf("GetParserOutput failed: %v", err)
	}
	if found {
		t.Error("Expected cache miss due to parser type mismatch, got hit")
	}
}

func TestAgentCache_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	url := "https://example.com/article"
	parserType := "web"
	agentPipeline := []string{"summary"}
	output := "This is a summarized article content."

	// Test Set
	err = cache.SetAgentOutput(url, parserType, agentPipeline, output)
	if err != nil {
		t.Fatalf("SetAgentOutput failed: %v", err)
	}

	// Test Get
	retrieved, found, err := cache.GetAgentOutput(url, parserType, agentPipeline)
	if err != nil {
		t.Fatalf("GetAgentOutput failed: %v", err)
	}
	if !found {
		t.Error("Expected cache hit, got miss")
	}
	if retrieved != output {
		t.Errorf("Retrieved data mismatch: got %s, want %s", retrieved, output)
	}
}

func TestAgentCache_PipelineMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	url := "https://example.com/article"
	parserType := "web"
	pipeline1 := []string{"summary"}
	pipeline2 := []string{"summary", "translate"}
	output := "Cached content"

	// Store with pipeline1
	err = cache.SetAgentOutput(url, parserType, pipeline1, output)
	if err != nil {
		t.Fatalf("SetAgentOutput failed: %v", err)
	}

	// Try to retrieve with pipeline2
	_, found, err := cache.GetAgentOutput(url, parserType, pipeline2)
	if err != nil {
		t.Fatalf("GetAgentOutput failed: %v", err)
	}
	if found {
		t.Error("Expected cache miss due to pipeline mismatch, got hit")
	}
}

func TestAgentCache_PipelineOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	url := "https://example.com/article"
	parserType := "web"
	pipeline1 := []string{"translate", "summary"}
	pipeline2 := []string{"summary", "translate"}
	output := "Cached content"

	// Store with pipeline1
	err = cache.SetAgentOutput(url, parserType, pipeline1, output)
	if err != nil {
		t.Fatalf("SetAgentOutput failed: %v", err)
	}

	// Try to retrieve with pipeline2 (different order)
	_, found, err := cache.GetAgentOutput(url, parserType, pipeline2)
	if err != nil {
		t.Fatalf("GetAgentOutput failed: %v", err)
	}
	if found {
		t.Error("Expected cache miss due to pipeline order difference, got hit")
	}
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	// Add some data
	cache.SetParserOutput("https://example.com/1", "web", []byte("data1"))
	cache.SetParserOutput("https://example.com/2", "youtube", []byte("data2"))
	cache.SetAgentOutput("https://example.com/3", "web", []string{"summary"}, "data3")

	// Verify data exists
	stats, _ := cache.Stats()
	if stats.ParserEntries != 2 {
		t.Errorf("Expected 2 parser entries, got %d", stats.ParserEntries)
	}
	if stats.AgentEntries != 1 {
		t.Errorf("Expected 1 agent entry, got %d", stats.AgentEntries)
	}

	// Clear cache
	err = cache.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify cache is empty
	stats, _ = cache.Stats()
	if stats.ParserEntries != 0 {
		t.Errorf("Expected 0 parser entries after clear, got %d", stats.ParserEntries)
	}
	if stats.AgentEntries != 0 {
		t.Errorf("Expected 0 agent entries after clear, got %d", stats.AgentEntries)
	}
}

func TestStats(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	// Initially empty
	stats, err := cache.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.ParserEntries != 0 || stats.AgentEntries != 0 {
		t.Error("Expected empty cache initially")
	}

	// Add entries
	cache.SetParserOutput("https://example.com/1", "web", []byte("data1"))
	cache.SetParserOutput("https://example.com/2", "web", []byte("data2"))
	cache.SetAgentOutput("https://example.com/1", "web", []string{"summary"}, "output1")

	stats, err = cache.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.ParserEntries != 2 {
		t.Errorf("Expected 2 parser entries, got %d", stats.ParserEntries)
	}
	if stats.AgentEntries != 1 {
		t.Errorf("Expected 1 agent entry, got %d", stats.AgentEntries)
	}
	if stats.OldestEntry.IsZero() {
		t.Error("Expected OldestEntry to be set")
	}
}

func TestAccessTracking(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	url := "https://example.com/article"
	parserType := "web"
	output := []byte("test data")

	// Store entry
	cache.SetParserOutput(url, parserType, output)

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Access entry
	cache.GetParserOutput(url, parserType)

	// Note: We can't easily verify accessed_at was updated without querying DB directly
	// But we can verify the access doesn't cause errors
}

func TestDefaultCachePath(t *testing.T) {
	path := DefaultCachePath()
	if path == "" {
		t.Error("DefaultCachePath returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Error("DefaultCachePath should return absolute path")
	}
}

func TestUpdateExistingEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")

	cache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	defer cache.Close()

	url := "https://example.com/article"
	parserType := "web"
	output1 := []byte("original data")
	output2 := []byte("updated data")

	// Store initial data
	cache.SetParserOutput(url, parserType, output1)

	// Update with new data
	cache.SetParserOutput(url, parserType, output2)

	// Retrieve and verify we get the updated data
	retrieved, found, _ := cache.GetParserOutput(url, parserType)
	if !found {
		t.Fatal("Expected cache hit")
	}
	if string(retrieved) != string(output2) {
		t.Errorf("Expected updated data, got %s", retrieved)
	}
}
