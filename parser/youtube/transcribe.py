#!/usr/bin/env python3
"""
YouTube video transcription script using yt-dlp and faster-whisper.
This script downloads audio from YouTube videos and transcribes them with timing information.
"""

import sys
import os
import json
import tempfile
import subprocess
from pathlib import Path


def ensure_dependencies():
    """Ensure yt-dlp and faster-whisper are installed."""
    try:
        import yt_dlp
    except ImportError:
        subprocess.check_call([sys.executable, "-m", "pip", "install", "yt-dlp"])
    
    try:
        import faster_whisper
    except ImportError:
        subprocess.check_call([sys.executable, "-m", "pip", "install", "faster-whisper"])


def download_audio(video_url, output_path):
    """Download audio from YouTube video using yt-dlp."""
    import yt_dlp
    
    ydl_opts = {
        'format': 'bestaudio/best',
        'outtmpl': str(output_path),
        'postprocessors': [{
            'key': 'FFmpegExtractAudio',
            'preferredcodec': 'wav',
        }],
        'quiet': True,
        'no_warnings': True,
    }
    
    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        info = ydl.extract_info(video_url, download=True)
        return info.get('title', 'Unknown Title')


def transcribe_audio(audio_path, model_size="base"):
    """Transcribe audio file using faster-whisper with timing information."""
    from faster_whisper import WhisperModel
    
    model = WhisperModel(model_size, device="cpu", compute_type="int8")
    
    segments, info = model.transcribe(audio_path, beam_size=5)
    
    transcription = {
        "title": "",
        "language": info.language,
        "segments": []
    }
    
    for segment in segments:
        transcription["segments"].append({
            "start": segment.start,
            "end": segment.end,
            "text": segment.text.strip()
        })
    
    return transcription


def main():
    if len(sys.argv) != 2:
        print("Usage: transcribe.py <youtube_url>", file=sys.stderr)
        sys.exit(1)
    
    video_url = sys.argv[1]
    
    try:
        # Ensure dependencies are installed
        print("Installing dependencies...", file=sys.stderr)
        ensure_dependencies()
        
        with tempfile.TemporaryDirectory() as temp_dir:
            # Download audio
            print("Downloading audio...", file=sys.stderr)
            audio_path = Path(temp_dir) / "audio.%(ext)s"
            title = download_audio(video_url, audio_path)
            
            # Find the actual audio file (yt-dlp adds extension)
            audio_files = list(Path(temp_dir).glob("audio.*"))
            if not audio_files:
                raise Exception("No audio file found after download")
            
            actual_audio_path = audio_files[0]
            print(f"Audio downloaded: {actual_audio_path}", file=sys.stderr)
            
            # Transcribe
            print("Starting transcription...", file=sys.stderr)
            transcription = transcribe_audio(actual_audio_path)
            transcription["title"] = title
            
            # Output as JSON
            print(json.dumps(transcription, indent=2))
            
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()