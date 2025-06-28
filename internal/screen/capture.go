package screen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const FOLDER_PERMS_READALL_WRITEOWN = 0755

func CaptureScreenshot() error {
	// GO uses arbitrarily but tastifully chosen values for datetime formatting
	now := time.Now().Format("2006-01-02T15:04:05.000")
	outputPath := fmt.Sprintf(".data/%s.jpg", now)

	// Ensure .data directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), FOLDER_PERMS_READALL_WRITEOWN); err != nil {
		return fmt.Errorf("failed to create .data directory: %w", err)
	}

	// Use macOS's built-in screenshot command
	cmd := exec.Command("screencapture", "-x", outputPath)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run screencapture: %w", err)
	}

	return nil
}
