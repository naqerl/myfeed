#!/usr/bin/env python3
"""
YouTube video transcription script using yt-dlp subtitle extraction.
This script extracts subtitles from YouTube videos using yt-dlp's built-in subtitle functionality.
"""

import sys
import os
import json
import tempfile
import subprocess
import re
from pathlib import Path


def ensure_dependencies():
    """Ensure yt-dlp is installed."""
    packages_to_check = [
        ("yt_dlp", "yt-dlp"),
    ]
    
    for module_name, package_name in packages_to_check:
        try:
            __import__(module_name)
        except ImportError:
            print(f"Installing {package_name}...", file=sys.stderr)
            subprocess.check_call([sys.executable, "-m", "pip", "install", package_name])


def parse_webvtt_time(time_str):
    """Parse WebVTT time format (HH:MM:SS.mmm) to seconds."""
    # Handle both HH:MM:SS.mmm and MM:SS.mmm formats
    parts = time_str.split(':')
    if len(parts) == 3:
        hours, minutes, seconds = parts
        total_seconds = int(hours) * 3600 + int(minutes) * 60 + float(seconds)
    elif len(parts) == 2:
        minutes, seconds = parts
        total_seconds = int(minutes) * 60 + float(seconds)
    else:
        total_seconds = float(parts[0])
    
    return total_seconds


def parse_webvtt_content(content):
    """Parse WebVTT subtitle content into segments."""
    segments = []
    lines = content.split('\n')
    
    i = 0
    while i < len(lines):
        line = lines[i].strip()
        
        # Skip empty lines and WEBVTT header
        if not line or line.startswith('WEBVTT') or line.startswith('NOTE'):
            i += 1
            continue
            
        # Look for timestamp lines (format: start --> end)
        if '-->' in line:
            # Parse timestamp
            timestamp_match = re.match(r'([0-9:.,]+)\s*-->\s*([0-9:.,]+)', line)
            if timestamp_match:
                start_time = parse_webvtt_time(timestamp_match.group(1).replace(',', '.'))
                end_time = parse_webvtt_time(timestamp_match.group(2).replace(',', '.'))
                
                # Collect subtitle text (may span multiple lines)
                i += 1
                text_lines = []
                while i < len(lines) and lines[i].strip() and '-->' not in lines[i]:
                    text_line = lines[i].strip()
                    # Remove HTML tags and clean up text
                    text_line = re.sub(r'<[^>]+>', '', text_line)
                    if text_line:
                        text_lines.append(text_line)
                    i += 1
                
                if text_lines:
                    segments.append({
                        "start": start_time,
                        "end": end_time,
                        "text": " ".join(text_lines)
                    })
                    
                continue
        
        i += 1
    
    return segments


def extract_subtitles(video_url, temp_dir):
    """Extract subtitles from YouTube video using yt-dlp."""
    import yt_dlp  # type: ignore
    
    # Set up output template for subtitles
    subtitle_template = str(Path(temp_dir) / "%(title)s.%(ext)s")
    
    print("Extracting video info and subtitles...", file=sys.stderr)
    
    # Use yt-dlp to extract subtitles only
    import os
    import contextlib
    
    # Redirect stdout and stderr to capture any unwanted output from yt-dlp
    with open(os.devnull, 'w') as devnull:
        with contextlib.redirect_stdout(devnull), contextlib.redirect_stderr(devnull):
            ydl = yt_dlp.YoutubeDL({  # type: ignore
                'writesubtitles': True,
                'writeautomaticsubs': True, 
                'subtitleslangs': ['en'],
                'skip_download': True,
                'outtmpl': subtitle_template,
                'quiet': True,
                'no_warnings': True,
                'no_color': True,
                'extract_flat': False,
            })
            
            info = ydl.extract_info(video_url, download=True)
    
    title = info.get('title', 'Unknown Title')
    language = info.get('language', 'en')
    
    # Find downloaded subtitle files
    subtitle_files = list(Path(temp_dir).glob("*.vtt"))
    
    if not subtitle_files:
        raise Exception("No subtitle files found. This video may not have subtitles available.")
    
    # Prefer manual subtitles over auto-generated ones
    manual_subs = [f for f in subtitle_files if '.en.' in f.name and not '.auto.' in f.name]
    auto_subs = [f for f in subtitle_files if '.auto.' in f.name or ('.en.' in f.name)]
    
    subtitle_file = manual_subs[0] if manual_subs else (auto_subs[0] if auto_subs else subtitle_files[0])
    
    print(f"Using subtitle file: {subtitle_file.name}", file=sys.stderr)
    
    # Read and parse subtitle content
    with open(subtitle_file, 'r', encoding='utf-8') as f:
        subtitle_content = f.read()
    
    segments = parse_webvtt_content(subtitle_content)
    
    return {
        "title": title,
        "language": language,
        "segments": segments
    }


def main():
    if len(sys.argv) != 2:
        print("Usage: transcribe.py <youtube_url>", file=sys.stderr)
        sys.exit(1)
    
    video_url = sys.argv[1]
    
    try:
        print("Installing dependencies...", file=sys.stderr)
        ensure_dependencies()
        
        with tempfile.TemporaryDirectory() as temp_dir:
            print("Extracting subtitles...", file=sys.stderr)
            transcription = extract_subtitles(video_url, temp_dir)
            
            print(json.dumps(transcription, indent=2))
            
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()