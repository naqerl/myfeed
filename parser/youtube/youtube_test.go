package youtube

import (
	"strings"
	"testing"
)

type testCase struct {
	name         string
	videoURL     string
	description  string
	expectMethod string // "subtitle", "whisper"
	minSegments  int
	shouldSkip   bool
}

func TestYouTubeParser(t *testing.T) {
	testCases := []testCase{
		{
			name:         "RickRoll_WithSubtitles",
			videoURL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			description:  "Rick Astley - Never Gonna Give You Up (has good subtitles)",
			expectMethod: "subtitle",
			minSegments:  30,
			shouldSkip:   false,
		},
		{
			name:         "TestVideo_MayNeedWhisper",
			videoURL:     "https://www.youtube.com/watch?v=jO9RSppTirQ",
			description:  "Original test video (may need whisper fallback)",
			expectMethod: "whisper",
			minSegments:  1, // Lower expectation since whisper might be slow
			shouldSkip:   false,
		},
	}

	parser, err := New()
	if err != nil {
		t.Fatalf("Failed to create YouTube parser: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip whisper tests in short mode to save time
			if testing.Short() && tc.expectMethod == "whisper" {
				t.Skip("Skipping whisper fallback test in short mode")
				return
			}

			t.Logf("Testing: %s (%s)", tc.description, tc.videoURL)

			response, err := parser.Parse(tc.videoURL)
			if err != nil {
				t.Fatalf("Failed to parse YouTube video: %v", err)
			}

			validateResponse(t, response, tc)
		})
	}
}

func validateResponse(t *testing.T, response interface{}, tc testCase) {
	result := response.(Response).String()
	if len(result) == 0 {
		t.Error("Empty response from YouTube parser")
	}

	// Test the parsed transcription structure
	resp, ok := response.(Response)
	if !ok {
		t.Fatal("Response is not of expected type")
	}

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
	if len(resp.Transcription.Segments) < tc.minSegments {
		t.Errorf("Expected at least %d segments, got %d", tc.minSegments, len(resp.Transcription.Segments))
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
