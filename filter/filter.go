package filter

import (
	"log/slog"
	"regexp"
	"strings"
	"unicode"

	"github.com/scipunch/myfeed/config"
	"github.com/scipunch/myfeed/fetcher/types"
)

// FilterPipeline applies a series of named filters to feed items
type FilterPipeline struct {
	filters map[string]*CompiledFilter
}

// CompiledFilter contains compiled regex patterns for efficient matching
type CompiledFilter struct {
	config          config.Filter
	excludePatterns []*regexp.Regexp
}

// NewFilterPipeline creates a new filter pipeline from config
func NewFilterPipeline(filtersConfig map[string]config.Filter) (*FilterPipeline, error) {
	compiled := make(map[string]*CompiledFilter)

	for name, filterCfg := range filtersConfig {
		cf := &CompiledFilter{
			config:          filterCfg,
			excludePatterns: make([]*regexp.Regexp, 0, len(filterCfg.ExcludePatterns)),
		}

		// Compile regex patterns
		for _, pattern := range filterCfg.ExcludePatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				slog.Warn("invalid regex pattern in filter", "filter", name, "pattern", pattern, "error", err)
				continue
			}
			cf.excludePatterns = append(cf.excludePatterns, re)
		}

		compiled[name] = cf
	}

	return &FilterPipeline{filters: compiled}, nil
}

// ShouldInclude returns true if the item passes all filters in the pipeline
// filterNames is a list of filter names to apply in order
func (fp *FilterPipeline) ShouldInclude(item types.FeedItem, filterNames []string) (bool, string) {
	if len(filterNames) == 0 {
		return true, "" // No filters = include everything
	}

	for _, filterName := range filterNames {
		filter, exists := fp.filters[filterName]
		if !exists {
			slog.Warn("filter not found, skipping", "filter_name", filterName)
			continue
		}

		if shouldInclude, reason := fp.applyFilter(item, filter, filterName); !shouldInclude {
			return false, reason
		}
	}

	return true, ""
}

// applyFilter applies a single filter to an item
func (fp *FilterPipeline) applyFilter(item types.FeedItem, filter *CompiledFilter, filterName string) (bool, string) {
	// Get the text to analyze (title + description)
	text := item.Title + " " + item.Description

	// 1. Check minimum length
	if filter.config.MinLength > 0 && len(text) < filter.config.MinLength {
		return false, filterName + ":min_length"
	}

	// 2. Check minimum word count
	if filter.config.MinWords > 0 {
		wordCount := countWords(text)
		if wordCount < filter.config.MinWords {
			return false, filterName + ":min_words"
		}
	}

	// 3. Check exclude patterns
	for i, pattern := range filter.excludePatterns {
		if pattern.MatchString(text) {
			return false, filterName + ":exclude_pattern[" + filter.config.ExcludePatterns[i] + "]"
		}
	}

	// 4. Check paragraph requirement
	if filter.config.RequireParagraphs {
		if !hasMultipleParagraphs(text) {
			return false, filterName + ":require_paragraphs"
		}
	}

	return true, ""
}

// countWords counts the number of words in text
func countWords(text string) int {
	words := 0
	inWord := false

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if !inWord {
				words++
				inWord = true
			}
		} else {
			inWord = false
		}
	}

	return words
}

// hasMultipleParagraphs checks if text has multiple paragraphs
func hasMultipleParagraphs(text string) bool {
	// Look for double newlines or multiple single newlines
	lines := strings.Split(text, "\n")
	nonEmptyLines := 0

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	return nonEmptyLines >= 2
}
