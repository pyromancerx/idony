package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pyromancer/idony/internal/config"
)

type TTSTool struct {
	conf *config.Config
}

func NewTTSTool(conf *config.Config) *TTSTool {
	return &TTSTool{conf: conf}
}

func (t *TTSTool) Name() string {
	return "tts"
}

func (t *TTSTool) Description() string {
	return "Converts text to speech using Flite. Input: text to speak. Output: path to generated WAV file."
}

func (t *TTSTool) Execute(ctx context.Context, input string) (string, error) {
	flite := t.conf.GetWithDefault("FLITE_BIN", "flite")
	voice := t.conf.GetWithDefault("FLITE_VOICE", "slt")
	
	tempDir := "temp_audio"
	os.MkdirAll(tempDir, 0755)
	
	outputPath := filepath.Join(tempDir, fmt.Sprintf("tts_%d.wav", os.Getpid()))
	
	// Use -voice flag
	cmd := exec.CommandContext(ctx, flite, "-voice", voice, "-t", input, "-o", outputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("flite failed: %v, output: %s", err, string(out))
	}

	return outputPath, nil
}

func (t *TTSTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Text to Speech",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Text", "type": "longtext", "required": true},
		},
	}
}
