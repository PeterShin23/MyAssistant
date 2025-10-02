package screen

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const FOLDER_PERMS_READALL_WRITEOWN = 0755

type DisplayConfig struct {
	Display int `json:"display"`
}

func CaptureScreenshot() (string, error) {
	// GO uses arbitrarily but tastifully chosen values for datetime formatting
	now := time.Now().Format("2006-01-02T15:04:05.000")
	outputPath := fmt.Sprintf(".data/%s.jpg", now)

	// Ensure .data directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), FOLDER_PERMS_READALL_WRITEOWN); err != nil {
		return "", fmt.Errorf("failed to create .data directory: %w", err)
	}

	// Default to display 1
	display := 1

	// Try to read from rules.json
	rulesBytes, err := os.ReadFile("rules.json")
	if err == nil {
		var cfg DisplayConfig
		if err := json.Unmarshal(rulesBytes, &cfg); err == nil && cfg.Display > 0 {
			display = cfg.Display
		}
	}

	// Use macOS's built-in screenshot command
	cmd := exec.Command("screencapture", "-x", "-D", fmt.Sprintf("%d", display), outputPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run screencapture: %w", err)
	}

	return outputPath, nil
}
