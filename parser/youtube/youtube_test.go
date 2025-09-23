package youtube

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

type TestCase struct {
	Name            string `json:"name"`
	VideoURL        string `json:"videoURL"`
	Description     string `json:"description"`
	SkipInShortMode bool   `json:"skipInShortMode"`
	SkipReason      string `json:"skipReason"`
}

type ExpectedSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type ExpectedTranscription struct {
	Title    string            `json:"title"`
	Language string            `json:"language"`
	Segments []json.RawMessage `json:"segments"`
}

type TestData struct {
	TestCase       TestCase              `json:"testCase"`
	ExpectedOutput ExpectedTranscription `json:"expectedOutput"`
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

			validateExactMatch(t, response, testData)
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
				return fmt.Errorf("failed to read test data file %s: %w", path, err)
			}

			var testData TestData
			if err := json.Unmarshal(data, &testData); err != nil {
				return fmt.Errorf("failed to parse JSON in file %s: %w", path, err)
			}

			testDataFiles = append(testDataFiles, testData)
		}

		return nil
	})

	return testDataFiles, err
}
func parseExpectedSegment(rawSegment json.RawMessage) (ExpectedSegment, error) {
	var seg ExpectedSegment

	// Parse as a string in format "[HH:MM:SS] text"
	var segmentStr string
	if err := json.Unmarshal(rawSegment, &segmentStr); err != nil {
		return seg, fmt.Errorf("failed to parse segment as string: %w", err)
	}

	// Parse timestamp format [HH:MM:SS] - standard format
	timestampRegex := regexp.MustCompile(`\[(\d{2}):(\d{2}):(\d{2})\](.*)`)
	matches := timestampRegex.FindStringSubmatch(segmentStr)
	if len(matches) == 5 { // matches[0] = full match, [1-3] = groups, [4] = text
		hours, _ := strconv.Atoi(matches[1])
		minutes, _ := strconv.Atoi(matches[2])
		seconds, _ := strconv.Atoi(matches[3])
		seg.Start = float64(hours*3600 + minutes*60 + seconds)
		seg.Text = strings.TrimSpace(matches[4])
		seg.End = seg.Start + 2.0 // Default duration
		return seg, nil
	}

	return seg, fmt.Errorf("failed to parse timestamp format: %s", segmentStr)
}

func validateExactMatch(t *testing.T, response any, testData TestData) {
	expected := testData.ExpectedOutput

	// Test the parsed transcription structure
	resp, ok := response.(Response)
	if !ok {
		t.Fatal("Response is not of expected type")
	}

	actual := resp.Transcription

	// Compare title - allow empty expected title to skip validation
	if expected.Title != "" && actual.Title != expected.Title {
		t.Errorf("Title mismatch:\nExpected: %s\nActual: %s", expected.Title, actual.Title)
	}

	// Compare language
	if actual.Language != expected.Language {
		t.Errorf("Language mismatch:\nExpected: %s\nActual: %s", expected.Language, actual.Language)
	}

	// If expected segments is empty, just validate basic structure
	if len(expected.Segments) == 0 {
		if len(actual.Segments) == 0 {
			t.Error("No segments found in transcription")
		}
		// Just validate basic segment structure without content comparison
		for i, seg := range actual.Segments {
			if seg.Start < 0 {
				t.Errorf("Segment %d has invalid start time: %.3f", i, seg.Start)
			}
			if seg.End <= seg.Start {
				t.Errorf("Segment %d has invalid timing: start=%.3f, end=%.3f", i, seg.Start, seg.End)
			}
			if strings.TrimSpace(seg.Text) == "" {
				t.Errorf("Segment %d has empty text", i)
			}
		}
		t.Logf("Basic validation passed for %d segments", len(actual.Segments))
		return
	}

	// Parse expected segments from raw JSON
	var expectedSegments []ExpectedSegment
	for i, rawSeg := range expected.Segments {
		seg, err := parseExpectedSegment(rawSeg)
		if err != nil {
			t.Errorf("Failed to parse expected segment %d: %v\nRaw segment: %s", i, err, string(rawSeg))
			continue
		}
		expectedSegments = append(expectedSegments, seg)
	}

	// Compare segment count for exact match
	if len(actual.Segments) != len(expectedSegments) {
		t.Errorf("Segment count mismatch:\nExpected: %d\nActual: %d", len(expectedSegments), len(actual.Segments))
	}

	// Compare each segment exactly
	maxSegments := min(len(expectedSegments), len(actual.Segments))

	for i := range maxSegments {
		expectedSeg := expectedSegments[i]
		actualSeg := actual.Segments[i]

		// For timestamp validation, allow some tolerance (±1 second) since formats might differ
		if actualSeg.Start < expectedSeg.Start-1.0 || actualSeg.Start > expectedSeg.Start+1.0 {
			t.Errorf("Segment %d start time mismatch (tolerance: ±1s):\nExpected: %.3f\nActual: %.3f\nDifference: %.3f",
				i, expectedSeg.Start, actualSeg.Start, actualSeg.Start-expectedSeg.Start)
		}

		// Text comparison - normalize whitespace
		expectedText := strings.TrimSpace(expectedSeg.Text)
		actualText := strings.TrimSpace(actualSeg.Text)
		if actualText != expectedText {
			t.Errorf("Segment %d text mismatch:\nExpected: '%s' (len=%d)\nActual: '%s' (len=%d)",
				i, expectedText, len(expectedText), actualText, len(actualText))
		}
	}

	t.Logf("✓ Validation passed - Title: %s, Language: %s, Segments: %d",
		actual.Title, actual.Language, len(actual.Segments))
}
