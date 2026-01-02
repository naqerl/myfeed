package config

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/BurntSushi/toml"

	"github.com/scipunch/myfeed/parser"
)

type ResourceType = string

var (
	RSS             = ResourceType("rss")
	TelegramChannel = ResourceType("telegram_channel")
)

const baseCfgPath = "myfeed/config.toml"

type Config struct {
	Resources       []ResourceConfig  `toml:"resources"`
	DatabasePath    string            `toml:"database_path"`
	OutputDirectory string            `toml:"output_directory"` // Directory for generated files (defaults to $HOME/myfeed)
	Filters         map[string]Filter `toml:"filters"`          // Named filters that can be referenced by resources
}

type ResourceConfig struct {
	FeedURL     string       `toml:"feed_url"`
	ParserT     parser.Type  `toml:"parser"`
	T           ResourceType `toml:"type"`
	Agents      []string     `toml:"agents"`  // Post-processing agents, e.g., ["summary"]
	Enabled     *bool        `toml:"enabled"` // Whether this resource is active (defaults to true if not set)
	FilterNames []string     `toml:"filters"` // Names of filters to apply (pipeline)
}

// Filter defines rules for filtering feed items
type Filter struct {
	MinLength         int      `toml:"min_length"`         // Minimum character count (0 = no limit)
	MinWords          int      `toml:"min_words"`          // Minimum word count (0 = no limit)
	ExcludePatterns   []string `toml:"exclude_patterns"`   // Regex patterns to exclude
	RequireParagraphs bool     `toml:"require_paragraphs"` // Must have multiple lines/paragraphs
}

// IsEnabled returns true if the resource is enabled (defaults to true if not explicitly set)
func (r ResourceConfig) IsEnabled() bool {
	if r.Enabled == nil {
		return true
	}
	return *r.Enabled
}

func Read(path string) (Config, error) {
	conf := Default()
	dat, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}
	_, err = toml.Decode(string(dat), &conf)
	if err != nil {
		return conf, fmt.Errorf("failed to decode config at %s with %w", path, err)
	}
	return conf, nil
}

func Write(cfgPath string, cfg Config) error {
	blob, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode config with %w", err)
	}
	basePath := path.Dir(cfgPath)
	err = os.MkdirAll(basePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create base config directory at '%s' with %w", basePath, err)
	}
	err = os.WriteFile(cfgPath, blob, 0644)
	if err != nil {
		return fmt.Errorf("failed to write into config file at '%s' with %w", cfgPath, err)
	}
	slog.Info("config written", "at", cfgPath)
	return nil
}

func Default() Config {
	var dbBase = path.Join(os.Getenv("HOME"), ".local/share/myfeed")
	var home = os.Getenv("HOME")
	var outputDir = path.Join(home, "myfeed")
	return Config{
		DatabasePath:    path.Join(dbBase, "data.db"),
		OutputDirectory: outputDir,
		Resources:       []ResourceConfig{},
	}
}

func DefaultPath() string {
	var xdgHome = os.Getenv("XDG_CONFIG_HOME")
	if xdgHome != "" {
		return path.Join(xdgHome, baseCfgPath)
	}

	var home = os.Getenv("HOME")
	if home != "" {
		return path.Join(home, ".config", baseCfgPath)
	}

	panic("unclear where to search for the config fie")
}
