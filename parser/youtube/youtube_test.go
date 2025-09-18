package youtube

import (
	"testing"
)

func TestYouTubeParser(t *testing.T) {
	parser, err := New()
	if err != nil {
		t.Fatalf("Failed to create YouTube parser: %v", err)
	}

	videoURL := "https://www.youtube.com/watch?v=jO9RSppTirQ"
	t.Logf("Testing YouTube parser with: %s", videoURL)

	response, err := parser.Parse(videoURL)
	if err != nil {
		t.Fatalf("Failed to parse YouTube video: %v", err)
	}

	result := response.String()
	if len(result) == 0 {
		t.Error("Empty response from YouTube parser")
	}

	t.Logf("Response length: %d characters", len(result))
	t.Logf("Response preview: %.200s...", result)
}
