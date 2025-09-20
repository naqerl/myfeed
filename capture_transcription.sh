#!/bin/bash
# Script to capture transcription data for test cases

if [ $# -ne 2 ]; then
    echo "Usage: $0 <video_url> <output_file>"
    exit 1
fi

VIDEO_URL="$1"
OUTPUT_FILE="$2"

echo "Capturing transcription for: $VIDEO_URL"
echo "Output will be saved to: $OUTPUT_FILE"

# Run the transcription script and save output
/tmp/myfeed_youtube_venv/bin/python parser/youtube/transcribe.py "$VIDEO_URL" 2>/dev/null > "$OUTPUT_FILE"

if [ $? -eq 0 ]; then
    echo "✓ Transcription captured successfully"
    echo "Segments found: $(cat "$OUTPUT_FILE" | jq '.segments | length')"
    echo "Title: $(cat "$OUTPUT_FILE" | jq -r '.title')"
    echo ""
    echo "To use this in a test case, copy the JSON content to your test data file:"
    echo "  cp $OUTPUT_FILE parser/youtube/_test_data/your_test_name.json"
else
    echo "✗ Failed to capture transcription"
    exit 1
fi