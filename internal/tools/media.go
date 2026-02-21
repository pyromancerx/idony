package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pyromancer/idony/internal/config"
	"github.com/pyromancer/idony/internal/db"
)

type TranscribeTool struct {
	conf  *config.Config
	store *db.Store
}

func NewTranscribeTool(conf *config.Config, store *db.Store) *TranscribeTool {
	return &TranscribeTool{conf: conf, store: store}
}

func (t *TranscribeTool) Name() string {
	return "transcribe"
}

func (t *TranscribeTool) Description() string {
	return `Transcribes YouTube videos or local audio files. 
Input: {"action": "youtube|file", "url": "youtube_url", "path": "local_path"}`
}

func (t *TranscribeTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action string `json:"action"`
		URL    string `json:"url"`
		Path   string `json:"path"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	var result string
	var err error

	switch req.Action {
	case "youtube":
		result, err = t.handleYouTube(ctx, req.URL)
	case "file":
		result, err = t.handleFile(ctx, req.Path)
	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}

	if err == nil && t.store != nil {
		// Index the result
		source := req.URL
		if source == "" { source = req.Path }
		t.store.SaveMediaIndex(source, result, "audio")
	}

	return result, err
}

func (t *TranscribeTool) handleYouTube(ctx context.Context, url string) (string, error) {
	ytdlp := t.conf.GetWithDefault("YTDLP_BIN", "yt-dlp")
	
	// 1. Try to get title and description first
	metaCmd := exec.CommandContext(ctx, ytdlp, "--get-title", "--get-description", url)
	metaOut, _ := metaCmd.CombinedOutput()

	// 2. Try to grab subs directly (much faster)
	tempDir, _ := os.MkdirTemp("", "idony-yt-*")
	defer os.RemoveAll(tempDir)

	subCmd := exec.CommandContext(ctx, ytdlp, "--skip-download", "--write-auto-subs", "--sub-lang", "en", "--sub-format", "srt", "-o", filepath.Join(tempDir, "sub"), url)
	subCmd.Run()

	files, _ := filepath.Glob(filepath.Join(tempDir, "*.srt"))
	if len(files) == 0 {
		files, _ = filepath.Glob(filepath.Join(tempDir, "*.vtt"))
	}

	if len(files) > 0 {
		content, err := os.ReadFile(files[0])
		if err == nil {
			return fmt.Sprintf("Metadata:\n%s\n\nTranscript (from subtitles):\n%s", string(metaOut), string(content)), nil
		}
	}

	// 3. Fallback: Download audio and use Whisper
	audioPath := filepath.Join(tempDir, "audio.wav")
	downloadCmd := exec.CommandContext(ctx, ytdlp, "-x", "--audio-format", "wav", "--postprocessor-args", "-ar 16000 -ac 1", "-o", audioPath, url)
	if err := downloadCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download audio for transcription: %w", err)
	}

	return t.transcribeAudio(ctx, audioPath)
}

func (t *TranscribeTool) handleFile(ctx context.Context, path string) (string, error) {
	ffmpeg := t.conf.GetWithDefault("FFMPEG_BIN", "ffmpeg")
	tempDir, _ := os.MkdirTemp("", "idony-audio-*")
	defer os.RemoveAll(tempDir)

	wavPath := filepath.Join(tempDir, "proc.wav")
	convCmd := exec.CommandContext(ctx, ffmpeg, "-i", path, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wavPath)
	if err := convCmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	return t.transcribeAudio(ctx, wavPath)
}

func (t *TranscribeTool) transcribeAudio(ctx context.Context, wavPath string) (string, error) {
	whisper := t.conf.Get("WHISPER_BIN")
	model := t.conf.Get("WHISPER_MODEL")

	if whisper == "" || model == "" {
		return "", fmt.Errorf("whisper binary or model not configured")
	}

	// Use Output() instead of CombinedOutput() to only get the transcribed text from stdout.
	// Technical logs and system info are typically sent to stderr.
	cmd := exec.CommandContext(ctx, whisper, "-m", model, "-f", wavPath, "-nt")
	output, err := cmd.Output()
	if err != nil {
		// If it fails, we check CombinedOutput just for the error message
		return fmt.Sprintf("Transcription failed: %v", err), nil
	}

	return t.cleanWhisperOutput(string(output)), nil
}

func (t *TranscribeTool) cleanWhisperOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines, technical logs, and whisper internal markers
		if trimmed == "" || 
		   strings.HasPrefix(trimmed, "[") || 
		   strings.HasPrefix(trimmed, "(") || 
		   strings.Contains(trimmed, "whisper_") ||
		   strings.Contains(trimmed, "system_info:") ||
		   strings.Contains(trimmed, "main: processing") {
			continue
		}
		result = append(result, trimmed)
	}
	
	if len(result) == 0 {
		return output
	}

	return strings.Join(result, " ")
}

func (t *TranscribeTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Transcription Manager",
		"actions": []map[string]interface{}{
			{
				"name":  "youtube",
				"label": "Transcribe YouTube",
				"fields": []map[string]interface{}{
					{"name": "url", "label": "YouTube URL", "type": "string", "required": true},
				},
			},
			{
				"name":  "file",
				"label": "Transcribe Local File",
				"fields": []map[string]interface{}{
					{"name": "path", "label": "File Path", "type": "string", "required": true},
				},
			},
		},
	}
}
