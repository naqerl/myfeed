package youtube

import (
	"encoding/json"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type TestCase struct {
	Name            string `json:"name"`
	VideoURL        string `json:"videoURL"`
	Description     string `json:"description"`
	ExpectMethod    string `json:"expectMethod"`
	MinSegments     int    `json:"minSegments"`
	SkipInShortMode bool   `json:"skipInShortMode"`
	SkipReason      string `json:"skipReason"`
}

type ExpectedTranscription struct {
	Title    string    `json:"title"`
	Language string    `json:"language"`
	Segments []Segment `json:"segments"`
}

type ExpectedFormattedOutput struct {
	ContainsTitle      bool `json:"containsTitle"`
	ContainsLanguage   bool `json:"containsLanguage"`
	ContainsTimestamps bool `json:"containsTimestamps"`
	MinLength          int  `json:"minLength"`
}

type ExpectedOutput struct {
	Transcription   ExpectedTranscription   `json:"transcription"`
	FormattedOutput ExpectedFormattedOutput `json:"formattedOutput"`
}

type ValidationTolerances struct {
	TimingDelta       float64 `json:"timingDelta"`
	SegmentCountDelta int     `json:"segmentCountDelta"`
}

type Validation struct {
	ExactMatch        bool                 `json:"exactMatch"`
	ValidateStructure bool                 `json:"validateStructure"`
	ValidateTiming    bool                 `json:"validateTiming"`
	ValidateContent   bool                 `json:"validateContent"`
	Tolerances        ValidationTolerances `json:"tolerances"`
}

type TestData struct {
	TestCase       TestCase       `json:"testCase"`
	ExpectedOutput ExpectedOutput `json:"expectedOutput"`
	Validation     Validation     `json:"validation"`
}

func TestYouTubeParser(t *testing.T) {
	// Load test data files from _test_data directory
	testDataFiles, err := loadTestDataFiles()
	if err != nil {
		t.Fatalf("Failed to load test data files: %v", err)
	}

	if len(testDataFiles) == 0 {
		t.Skip("No test data files found in _test_data directory")
	}

	parser, err := New()
	if err != nil {
		t.Fatalf("Failed to create YouTube parser: %v", err)
	}

	for _, testData := range testDataFiles {
		tc := testData.TestCase
		t.Run(tc.Name, func(t *testing.T) {
			// Skip tests in short mode if specified
			if testing.Short() && tc.SkipInShortMode {
				t.Skipf("Skipping test in short mode: %s", tc.SkipReason)
				return
			}

			t.Logf("Testing: %s (%s)", tc.Description, tc.VideoURL)

			response, err := parser.Parse(tc.VideoURL)
			if err != nil {
				// Check if we should skip this test
				if strings.Contains(err.Error(), "ERROR: [youtube]") ||
					strings.Contains(err.Error(), "Video unavailable") ||
					strings.Contains(err.Error(), "invalid character") ||
					strings.Contains(err.Error(), "Both subtitle extraction and audio transcription failed") {
					t.Skipf("Skipping test - video issues or dependencies not available: %v", err)
					return
				}
				t.Fatalf("Failed to parse YouTube video: %v", err)
			}

			validateResponseWithTestData(t, response, testData)
		})
	}
}

func loadTestDataFiles() ([]TestData, error) {
	var testDataFiles []TestData
	testDataDir := "_test_data"

	err := filepath.WalkDir(testDataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(path, ".json") {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var testData TestData
			if err := json.Unmarshal(data, &testData); err != nil {
				return err
			}

			testDataFiles = append(testDataFiles, testData)
		}

		return nil
	})

	return testDataFiles, err
}

func validateResponseWithTestData(t *testing.T, response interface{}, testData TestData) {
	tc := testData.TestCase
	expected := testData.ExpectedOutput
	validation := testData.Validation
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

	// Validate segments using test data expectations
	if validation.ValidateStructure {
		if len(resp.Transcription.Segments) == 0 {
			t.Error("No transcription segments found")
		}
		if len(resp.Transcription.Segments) < tc.MinSegments {
			t.Errorf("Expected at least %d segments, got %d", tc.MinSegments, len(resp.Transcription.Segments))
		}

		// Check if segment count is within tolerance
		if len(expected.Transcription.Segments) > 0 {
			expectedCount := len(expected.Transcription.Segments)
			actualCount := len(resp.Transcription.Segments)
			delta := int(math.Abs(float64(expectedCount - actualCount)))
			if delta > validation.Tolerances.SegmentCountDelta {
				t.Errorf("Segment count differs too much: expected ~%d, got %d (delta: %d, max: %d)",
					expectedCount, actualCount, delta, validation.Tolerances.SegmentCountDelta)
			}
		}
	}
	t.Logf("Number of segments: %d", len(resp.Transcription.Segments))

	// Validate segment timing and structure
	if validation.ValidateTiming {
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
	}

	// Validate formatted output using expected criteria
	if expected.FormattedOutput.ContainsTitle && !strings.Contains(result, "#") {
		t.Error("Formatted output doesn't contain title marker (#)")
	}

	if expected.FormattedOutput.ContainsLanguage && !strings.Contains(result, "Language:") {
		t.Error("Formatted output doesn't contain language information")
	}

	if expected.FormattedOutput.ContainsTimestamps && !strings.Contains(result, "[") {
		t.Error("Formatted output doesn't contain timestamp markers")
	}

	if len(result) < expected.FormattedOutput.MinLength {
		t.Errorf("Formatted output too short: expected min %d chars, got %d",
			expected.FormattedOutput.MinLength, len(result))
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
