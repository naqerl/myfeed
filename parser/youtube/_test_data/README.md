# YouTube Parser Test Data

This directory contains JSON test case files for the YouTube parser. Each JSON file defines a test case with expected outputs and validation criteria.

## Test Data Format

```json
{
  "testCase": {
    "name": "TestName",
    "videoURL": "https://www.youtube.com/watch?v=VIDEO_ID",
    "description": "Description of what this test validates",
    "expectMethod": "subtitle|whisper|either",
    "minSegments": 10,
    "skipInShortMode": false,
    "skipReason": "Reason for skipping in short mode"
  },
  "expectedOutput": {
    "transcription": {
      "title": "Expected video title (empty for any)",
      "language": "en",
      "segments": []
    },
    "formattedOutput": {
      "containsTitle": true,
      "containsLanguage": true,
      "containsTimestamps": true,
      "minLength": 1000
    }
  },
  "validation": {
    "exactMatch": false,
    "validateStructure": true,
    "validateTiming": true,
    "validateContent": false,
    "tolerances": {
      "timingDelta": 1.0,
      "segmentCountDelta": 10
    }
  }
}
```

## Field Descriptions

### testCase
- `name`: Unique test name (used as subtest name)
- `videoURL`: YouTube video URL to test
- `description`: Human-readable test description
- `expectMethod`: Expected transcription method ("subtitle", "whisper", or "either")
- `minSegments`: Minimum number of segments expected
- `skipInShortMode`: Whether to skip this test when running with `-short` flag
- `skipReason`: Explanation for why test is skipped in short mode

### expectedOutput.transcription
- `title`: Expected video title (empty string means any title is acceptable)
- `language`: Expected language code
- `segments`: Array of expected segments (empty array means validate structure only)

### expectedOutput.formattedOutput
- `containsTitle`: Formatted output should contain title marker (`#`)
- `containsLanguage`: Formatted output should contain "Language:" text
- `containsTimestamps`: Formatted output should contain timestamp markers (`[mm:ss]`)
- `minLength`: Minimum character count for formatted output

### validation
- `exactMatch`: Whether to require exact match of transcription (usually false)
- `validateStructure`: Validate basic structure (title, language, segments)
- `validateTiming`: Validate segment timing (start < end, positive values)
- `validateContent`: Validate specific content matches (usually false for flexibility)
- `tolerances.timingDelta`: Acceptable difference in segment timing (seconds)
- `tolerances.segmentCountDelta`: Acceptable difference in segment count

## Adding New Test Cases

1. Create a new `.json` file in this directory
2. Use the format above with appropriate test data
3. Run tests to verify the new case works: `go test -v ./parser/youtube/`

## Notes

- Tests with `skipInShortMode: true` will be skipped when running `go test -short`
- Use `expectMethod: "either"` for videos that might use subtitle or whisper transcription
- Keep `exactMatch: false` unless you need exact content validation
- Use reasonable tolerances to account for platform variations