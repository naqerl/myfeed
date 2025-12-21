package youtube

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/scipunch/myfeed/fetcher/types"
	"github.com/scipunch/myfeed/parser"
)

//go:embed transcribe.py
var transcribeScript string

type Parser struct {
	venvPath   string
	pythonPath string
}

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type Transcription struct {
	Title    string    `json:"title"`
	Language string    `json:"language"`
	Segments []Segment `json:"segments"`
}

type Response struct {
	Transcription Transcription
}

func (r Response) String() string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("# %s\n\n", r.Transcription.Title))
	result.WriteString(fmt.Sprintf("**Language:** %s\n\n", r.Transcription.Language))
	result.WriteString("## Transcription\n\n")

	for _, segment := range r.Transcription.Segments {
		minutes := int(segment.Start) / 60
		seconds := int(segment.Start) % 60
		result.WriteString(fmt.Sprintf("[%02d:%02d] %s\n\n", minutes, seconds, segment.Text))
	}

	return result.String()
}

func New() (Parser, error) {
	var p Parser

	slog.Info("youtube parser: initializing")

	// Set up virtual environment path in temp directory
	tempDir := os.TempDir()
	p.venvPath = filepath.Join(tempDir, "myfeed_youtube_venv")

	// Determine Python executable path
	if isWindows() {
		p.pythonPath = filepath.Join(p.venvPath, "Scripts", "python.exe")
	} else {
		p.pythonPath = filepath.Join(p.venvPath, "bin", "python")
	}

	slog.Info("youtube parser: setting up virtual environment", "path", p.venvPath)

	// Create virtual environment if it doesn't exist
	if err := p.ensureVirtualEnv(); err != nil {
		return p, fmt.Errorf("failed to set up virtual environment: %w", err)
	}

	slog.Info("youtube parser: initialization complete")
	return p, nil
}

func (p Parser) ensureVirtualEnv() error {
	// Check if virtual environment exists
	if _, err := os.Stat(p.pythonPath); err == nil {
		slog.Info("youtube parser: virtual environment already exists")
		return nil // Virtual environment already exists
	}

	slog.Info("youtube parser: creating virtual environment")

	// Create virtual environment
	cmd := exec.Command("python3", "-m", "venv", p.venvPath)
	if err := cmd.Run(); err != nil {
		slog.Info("youtube parser: python3 failed, trying python")
		// Try with python if python3 is not available
		cmd = exec.Command("python", "-m", "venv", p.venvPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create virtual environment: %w", err)
		}
	}

	slog.Info("youtube parser: virtual environment created successfully")
	return nil
}

func (p Parser) Parse(item types.FeedItem) (parser.Response, error) {
	var resp Response

	slog.Info("youtube parser: starting transcription", "url", item.Link)

	// Create temporary script file
	scriptPath := filepath.Join(p.venvPath, "transcribe.py")
	if err := os.WriteFile(scriptPath, []byte(transcribeScript), 0755); err != nil {
		return resp, fmt.Errorf("failed to write transcribe script: %w", err)
	}
	defer os.Remove(scriptPath)

	slog.Info("youtube parser: executing transcription script")

	// Execute transcription script
	cmd := exec.Command(p.pythonPath, scriptPath, item.Link)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			slog.Error("youtube parser: transcription failed", "error", string(exitErr.Stderr))
			return resp, fmt.Errorf("transcription failed: %s", string(exitErr.Stderr))
		}
		return resp, fmt.Errorf("failed to execute transcription: %w", err)
	}

	slog.Info("youtube parser: parsing transcription output")

	// Parse JSON output
	if err := json.Unmarshal(output, &resp.Transcription); err != nil {
		return resp, fmt.Errorf("failed to parse transcription output: %w", err)
	}

	slog.Info("youtube parser: transcription completed", "title", resp.Transcription.Title, "segments", len(resp.Transcription.Segments))

	return resp, nil
}

func isWindows() bool {
	return strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")
}
