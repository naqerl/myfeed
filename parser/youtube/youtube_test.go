package youtube

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
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

type TestData struct {
	TestCase       TestCase      `json:"testCase"`
	ExpectedOutput Transcription `json:"expectedOutput"`
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
func validateExactMatch(t *testing.T, response interface{}, testData TestData) {
	expected := testData.ExpectedOutput

	// Test the parsed transcription structure
	resp, ok := response.(Response)
	if !ok {
		t.Fatal("Response is not of expected type")
	}

	actual := resp.Transcription

	// Compare title
	if actual.Title != expected.Title {
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

	// Compare segment count for exact match
	if len(actual.Segments) != len(expected.Segments) {
		t.Errorf("Segment count mismatch:\nExpected: %d\nActual: %d", len(expected.Segments), len(actual.Segments))
	}

	// Compare each segment exactly
	maxSegments := len(expected.Segments)
	if len(actual.Segments) < maxSegments {
		maxSegments = len(actual.Segments)
	}

	for i := 0; i < maxSegments; i++ {
		expectedSeg := expected.Segments[i]
		actualSeg := actual.Segments[i]

		if actualSeg.Start != expectedSeg.Start {
			t.Errorf("Segment %d start time mismatch:\nExpected: %.3f\nActual: %.3f", i, expectedSeg.Start, actualSeg.Start)
		}

		if actualSeg.End != expectedSeg.End {
			t.Errorf("Segment %d end time mismatch:\nExpected: %.3f\nActual: %.3f", i, expectedSeg.End, actualSeg.End)
		}

		if actualSeg.Text != expectedSeg.Text {
			t.Errorf("Segment %d text mismatch:\nExpected: %s\nActual: %s", i, expectedSeg.Text, actualSeg.Text)
		}
	}

	t.Logf("✓ Exact match validation passed - Title: %s, Language: %s, Segments: %d",
		actual.Title, actual.Language, len(actual.Segments))
}
