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
    """Ensure yt-dlp, faster-whisper and tqdm are installed."""
    # First ensure tqdm is available
    try:
        import tqdm
    except ImportError:
        print("Installing tqdm...", file=sys.stderr)
        subprocess.check_call([sys.executable, "-m", "pip", "install", "tqdm"])
    
    from tqdm import tqdm as progress_bar
    
    packages_to_check = [
        ("yt_dlp", "yt-dlp"),
        ("faster_whisper", "faster-whisper")
    ]
    
    for module_name, package_name in progress_bar(packages_to_check, desc="Checking dependencies"):
        try:
            __import__(module_name)
        except ImportError:
            print(f"Installing {package_name}...", file=sys.stderr)
            subprocess.check_call([sys.executable, "-m", "pip", "install", package_name])


def download_audio(video_url, output_path):
    """Download audio from YouTube video using yt-dlp."""
    import yt_dlp
    from tqdm import tqdm
    
    class TqdmProgressHook:
        def __init__(self):
            self.pbar = None
            
        def __call__(self, d):
            if d['status'] == 'downloading':
                if self.pbar is None:
                    total = d.get('total_bytes') or d.get('total_bytes_estimate')
                    if total:
                        self.pbar = tqdm(total=total, unit='B', unit_scale=True, desc="Downloading audio")
                
                if self.pbar and 'downloaded_bytes' in d:
                    self.pbar.update(d['downloaded_bytes'] - self.pbar.n)
                    
            elif d['status'] == 'finished':
                if self.pbar:
                    self.pbar.close()
                print("Download completed, extracting audio...", file=sys.stderr)
    
    progress_hook = TqdmProgressHook()
    
    ydl_opts = {
        'format': 'bestaudio/best',
        'outtmpl': str(output_path),
        'postprocessors': [{
            'key': 'FFmpegExtractAudio',
            'preferredcodec': 'wav',
        }],
        'quiet': True,
        'no_warnings': True,
        'progress_hooks': [progress_hook],
    }
    
    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        info = ydl.extract_info(video_url, download=True)
        return info.get('title', 'Unknown Title')


def transcribe_audio(audio_path, model_size="base"):
    """Transcribe audio file using faster-whisper with timing information."""
    from faster_whisper import WhisperModel
    from tqdm import tqdm
    
    print("Loading Whisper model...", file=sys.stderr)
    model = WhisperModel(model_size, device="cpu", compute_type="int8")
    
    print("Starting transcription...", file=sys.stderr)
    segments, info = model.transcribe(audio_path, beam_size=5)
    
    transcription = {
        "title": "",
        "language": info.language,
        "segments": []
    }
    
    # Convert segments to list first to get progress bar
    segments_list = list(segments)
    
    for segment in tqdm(segments_list, desc="Processing segments"):
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
        from tqdm import tqdm
        
        # Overall progress tracking
        stages = [
            "Installing dependencies",
            "Downloading audio", 
            "Transcribing audio",
            "Finalizing output"
        ]
        
        with tqdm(total=len(stages), desc="Overall progress") as pbar:
            # Ensure dependencies are installed
            pbar.set_description("Installing dependencies")
            ensure_dependencies()
            pbar.update(1)
            
            with tempfile.TemporaryDirectory() as temp_dir:
                # Download audio
                pbar.set_description("Downloading audio")
                audio_path = Path(temp_dir) / "audio.%(ext)s"
                title = download_audio(video_url, audio_path)
                pbar.update(1)
                
                # Find the actual audio file (yt-dlp adds extension)
                audio_files = list(Path(temp_dir).glob("audio.*"))
                if not audio_files:
                    raise Exception("No audio file found after download")
                
                actual_audio_path = audio_files[0]
                print(f"Audio downloaded: {actual_audio_path}", file=sys.stderr)
                
                # Transcribe
                pbar.set_description("Transcribing audio")
                transcription = transcribe_audio(actual_audio_path)
                transcription["title"] = title
                pbar.update(1)
                
                # Output as JSON
                pbar.set_description("Finalizing output")
                print(json.dumps(transcription, indent=2))
                pbar.update(1)
                
    except ImportError:
        # Fallback if tqdm is not available yet
        print("Installing dependencies...", file=sys.stderr)
        ensure_dependencies()
        
        with tempfile.TemporaryDirectory() as temp_dir:
            print("Downloading audio...", file=sys.stderr)
            audio_path = Path(temp_dir) / "audio.%(ext)s"
            title = download_audio(video_url, audio_path)
            
            audio_files = list(Path(temp_dir).glob("audio.*"))
            if not audio_files:
                raise Exception("No audio file found after download")
            
            actual_audio_path = audio_files[0]
            print(f"Audio downloaded: {actual_audio_path}", file=sys.stderr)
            
            print("Starting transcription...", file=sys.stderr)
            transcription = transcribe_audio(actual_audio_path)
            transcription["title"] = title
            
            print(json.dumps(transcription, indent=2))
            
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()