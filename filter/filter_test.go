package filter

import (
	"testing"
	"time"

	"github.com/scipunch/myfeed/config"
	"github.com/scipunch/myfeed/fetcher/types"
)

func TestFilterPipeline_MinLength(t *testing.T) {
	filters := map[string]config.Filter{
		"short": {
			MinLength: 50,
		},
	}

	pipeline, err := NewFilterPipeline(filters)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	tests := []struct {
		name          string
		item          types.FeedItem
		filterNames   []string
		shouldInclude bool
	}{
		{
			name: "long enough",
			item: types.FeedItem{
				Title:       "Test Title",
				Description: "This is a long enough description that should pass the filter",
			},
			filterNames:   []string{"short"},
			shouldInclude: true,
		},
		{
			name: "too short",
			item: types.FeedItem{
				Title:       "Short",
				Description: "Too short",
			},
			filterNames:   []string{"short"},
			shouldInclude: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			include, _ := pipeline.ShouldInclude(tt.item, tt.filterNames)
			if include != tt.shouldInclude {
				t.Errorf("Expected shouldInclude=%v, got %v", tt.shouldInclude, include)
			}
		})
	}
}

func TestFilterPipeline_MinWords(t *testing.T) {
	filters := map[string]config.Filter{
		"word_count": {
			MinWords: 10,
		},
	}

	pipeline, err := NewFilterPipeline(filters)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	tests := []struct {
		name          string
		item          types.FeedItem
		shouldInclude bool
	}{
		{
			name: "enough words",
			item: types.FeedItem{
				Title:       "Test Article",
				Description: "This is a description with enough words to pass the filter test successfully",
			},
			shouldInclude: true,
		},
		{
			name: "too few words",
			item: types.FeedItem{
				Title:       "Short",
				Description: "Not enough words",
			},
			shouldInclude: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			include, _ := pipeline.ShouldInclude(tt.item, []string{"word_count"})
			if include != tt.shouldInclude {
				t.Errorf("Expected shouldInclude=%v, got %v", tt.shouldInclude, include)
			}
		})
	}
}

func TestFilterPipeline_ExcludePatterns(t *testing.T) {
	filters := map[string]config.Filter{
		"russian_announcements": {
			ExcludePatterns: []string{
				"^[Дд]ержите.*",
				"^[Сс]мотрите.*",
				"^[Вв]прочем.*",
			},
		},
	}

	pipeline, err := NewFilterPipeline(filters)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	tests := []struct {
		name          string
		item          types.FeedItem
		shouldInclude bool
	}{
		{
			name: "normal content",
			item: types.FeedItem{
				Title:       "Interesting article",
				Description: "This is a normal article about something interesting",
			},
			shouldInclude: true,
		},
		{
			name: "starts with 'Держите'",
			item: types.FeedItem{
				Title:       "Держите комикс",
				Description: "Some content",
			},
			shouldInclude: false,
		},
		{
			name: "starts with 'держите' lowercase",
			item: types.FeedItem{
				Title:       "держите ссылку",
				Description: "Link here",
			},
			shouldInclude: false,
		},
		{
			name: "starts with 'Смотрите'",
			item: types.FeedItem{
				Title:       "Смотрите видео",
				Description: "Video link",
			},
			shouldInclude: false,
		},
		{
			name: "starts with 'Впрочем'",
			item: types.FeedItem{
				Title:       "Впрочем комикс тоже держите",
				Description: "Comic content",
			},
			shouldInclude: false,
		},
		{
			name: "contains but doesn't start with pattern",
			item: types.FeedItem{
				Title:       "Interesting article and держите link",
				Description: "Article content",
			},
			shouldInclude: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			include, reason := pipeline.ShouldInclude(tt.item, []string{"russian_announcements"})
			if include != tt.shouldInclude {
				t.Errorf("Expected shouldInclude=%v, got %v (reason: %s)", tt.shouldInclude, include, reason)
			}
		})
	}
}

func TestFilterPipeline_RequireParagraphs(t *testing.T) {
	filters := map[string]config.Filter{
		"paragraphs": {
			RequireParagraphs: true,
		},
	}

	pipeline, err := NewFilterPipeline(filters)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	tests := []struct {
		name          string
		item          types.FeedItem
		shouldInclude bool
	}{
		{
			name: "multiple paragraphs",
			item: types.FeedItem{
				Title:       "Article Title",
				Description: "First paragraph with some content.\n\nSecond paragraph with more content.",
			},
			shouldInclude: true,
		},
		{
			name: "single line",
			item: types.FeedItem{
				Title:       "Short announcement",
				Description: "Just one line of text",
			},
			shouldInclude: false,
		},
		{
			name: "multiple lines",
			item: types.FeedItem{
				Title:       "Title",
				Description: "First line\nSecond line\nThird line",
			},
			shouldInclude: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			include, _ := pipeline.ShouldInclude(tt.item, []string{"paragraphs"})
			if include != tt.shouldInclude {
				t.Errorf("Expected shouldInclude=%v, got %v", tt.shouldInclude, include)
			}
		})
	}
}

func TestFilterPipeline_Pipeline(t *testing.T) {
	filters := map[string]config.Filter{
		"length": {
			MinLength: 30,
		},
		"words": {
			MinWords: 5,
		},
		"patterns": {
			ExcludePatterns: []string{"^[Дд]ержите.*"},
		},
	}

	pipeline, err := NewFilterPipeline(filters)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Test that filters are applied in order (pipeline)
	item := types.FeedItem{
		Title:       "Держите длинную статью с большим количеством слов",
		Description: "This is a longer description",
		Published:   time.Now(),
	}

	// Should pass length and word filters but fail pattern filter
	include, reason := pipeline.ShouldInclude(item, []string{"length", "words", "patterns"})
	if include {
		t.Errorf("Expected item to be filtered out by patterns, but it passed")
	}
	if reason != "patterns:exclude_pattern[^[Дд]ержите.*]" {
		t.Errorf("Expected reason to mention pattern filter, got: %s", reason)
	}
}

func TestFilterPipeline_NoFilters(t *testing.T) {
	pipeline, err := NewFilterPipeline(map[string]config.Filter{})
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	item := types.FeedItem{
		Title:       "Any title",
		Description: "Any content",
	}

	// With no filters specified, should include everything
	include, _ := pipeline.ShouldInclude(item, []string{})
	if !include {
		t.Errorf("Expected item to be included when no filters applied")
	}
}
