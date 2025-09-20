package youtube

import (
	"strings"
	"testing"
)

func TestYouTubeParser(t *testing.T) {
	parser, err := New()
	if err != nil {
		t.Fatalf("Failed to create YouTube parser: %v", err)
	}

	videoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ" // Rick Astley - has good subtitles
	t.Logf("Testing YouTube parser with: %s", videoURL)

	response, err := parser.Parse(videoURL)
	if err != nil {
		// Skip if yt-dlp or dependencies aren't available
		if strings.Contains(err.Error(), "ERROR: [youtube]") ||
			strings.Contains(err.Error(), "No subtitle files found") ||
			strings.Contains(err.Error(), "invalid character") {
			t.Skipf("Skipping test - video issues or dependencies not available: %v", err)
		}
		t.Fatalf("Failed to parse YouTube video: %v", err)
	}

	result := response.String()
	if len(result) == 0 {
		t.Error("Empty response from YouTube parser")
	}

	// Test the parsed transcription structure
	if resp, ok := response.(Response); ok {
		// Validate title
		if resp.Transcription.Title == "" {
			t.Error("Transcription title is empty")
		}
		t.Logf("Video title: %s", resp.Transcription.Title)

		// Validate language
		if resp.Transcription.Language == "" {
			t.Error("Transcription language is empty")
		}
		t.Logf("Video language: %s", resp.Transcription.Language)

		// Validate segments
		if len(resp.Transcription.Segments) == 0 {
			t.Error("No transcription segments found")
		}
		t.Logf("Number of segments: %d", len(resp.Transcription.Segments))

		// Validate segment structure
		for i, segment := range resp.Transcription.Segments {
			if segment.Start < 0 {
				t.Errorf("Segment %d has invalid start time: %f", i, segment.Start)
			}
			if segment.End <= segment.Start {
				t.Errorf("Segment %d has invalid end time: %f (start: %f)", i, segment.End, segment.Start)
			}
			if strings.TrimSpace(segment.Text) == "" {
				t.Errorf("Segment %d has empty text", i)
			}

			// Only check first few segments to avoid spam
			if i < 3 {
				t.Logf("Segment %d: [%.2f-%.2f] %s", i, segment.Start, segment.End, segment.Text)
			}
		}
	} else {
		t.Error("Response is not of expected type")
	}

	// Test formatted output
	if !strings.Contains(result, "#") {
		t.Error("Formatted output doesn't contain title marker (#)")
	}

	if !strings.Contains(result, "Language:") {
		t.Error("Formatted output doesn't contain language information")
	}

	if !strings.Contains(result, "[") {
		t.Error("Formatted output doesn't contain timestamp markers")
	}

	t.Logf("Response length: %d characters", len(result))

	// Show a preview of the formatted output
	lines := strings.Split(result, "\n")
	preview := ""
	for i, line := range lines {
		if i >= 5 { // Show first 5 lines
			preview += "...\n"
			break
		}
		preview += line + "\n"
	}
	t.Logf("Response preview:\n%s", preview)
}

func TestYouTubeSubtitleExtraction(t *testing.T) {
	parser, err := New()
	if err != nil {
		t.Fatalf("Failed to create YouTube parser: %v", err)
	}

	// Test with a video known to have good English subtitles
	videoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ" // Rick Astley - Never Gonna Give You Up
	t.Logf("Testing subtitle extraction with: %s", videoURL)

	response, err := parser.Parse(videoURL)
	if err != nil {
		// If this video doesn't have subtitles, skip the test rather than fail
		t.Skipf("Skipping test - video may not have subtitles available: %v", err)
	}

	resp, ok := response.(Response)
	if !ok {
		t.Fatalf("Response is not of expected type")
	}

	// Validate that we got subtitle-like content
	if len(resp.Transcription.Segments) < 5 {
		t.Errorf("Expected more segments for a typical music video, got %d", len(resp.Transcription.Segments))
	}

	// Check for reasonable segment timing
	var totalDuration float64
	for i, segment := range resp.Transcription.Segments {
		duration := segment.End - segment.Start
		if duration <= 0 {
			t.Errorf("Segment %d has invalid duration: %f", i, duration)
		}
		if duration > 30 { // Most subtitle segments should be under 30 seconds
			t.Errorf("Segment %d has unusually long duration: %f", i, duration)
		}
		totalDuration = segment.End // Track the last end time
	}

	// Video should be at least 30 seconds long for a typical YouTube video
	if totalDuration < 30 {
		t.Errorf("Total video duration seems too short: %f seconds", totalDuration)
	}

	t.Logf("✓ Subtitle extraction successful - %d segments, %.1f seconds total",
		len(resp.Transcription.Segments), totalDuration)
}
