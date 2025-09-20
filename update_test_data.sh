#!/bin/bash
# Script to update test data with full transcription

VIDEO_URL="https://www.youtube.com/watch?v=dQw4w9WgXcQ"
TEST_FILE="parser/youtube/_test_data/rick_roll_subtitles.json"

echo "Capturing full transcription..."
TRANSCRIPTION=$(/tmp/myfeed_youtube_venv/bin/python parser/youtube/transcribe.py "$VIDEO_URL" 2>/dev/null)

if [ $? -eq 0 ]; then
    # Extract title, language, and segments
    TITLE=$(echo "$TRANSCRIPTION" | jq -r '.title')
    LANGUAGE=$(echo "$TRANSCRIPTION" | jq -r '.language')
    SEGMENTS=$(echo "$TRANSCRIPTION" | jq '.segments')
    
    # Create updated test data
    cat > "$TEST_FILE" << EOF
{
  "testCase": {
    "name": "RickRoll_WithSubtitles",
    "videoURL": "$VIDEO_URL",
    "description": "Rick Astley video with full transcription validation",
    "skipInShortMode": false,
    "skipReason": ""
  },
  "expectedOutput": {
    "title": "$TITLE",
    "language": "$LANGUAGE",
    "segments": $SEGMENTS
  }
}
EOF

    echo "✓ Test data updated with full transcription"
    echo "Segments: $(echo "$SEGMENTS" | jq 'length')"
    echo ""
    echo "You can now run: go test -v ./parser/youtube/"
else
    echo "✗ Failed to capture transcription"
fi