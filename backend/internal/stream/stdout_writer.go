package stream

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
)

// StdoutWriter implements StreamWriter for writing to stdout
type StdoutWriter struct {
	pretty      bool
	fullContent string
}

// NewStdoutWriter creates a new StdoutWriter
func NewStdoutWriter(pretty bool) *StdoutWriter {
	return &StdoutWriter{
		pretty: pretty,
	}
}

// WriteChunk writes a chunk to stdout
func (w *StdoutWriter) WriteChunk(chunk string) error {
	if !w.pretty {
		// For raw mode, write chunks as they arrive
		_, err := fmt.Fprint(os.Stdout, chunk)
		return err
	}
	// For pretty mode, accumulate the full content
	w.fullContent += chunk
	return nil
}

// MarkStreamComplete marks the current stream as complete for stdout writer
func (w *StdoutWriter) MarkStreamComplete() error {
	// For stdout, we don't need to do anything special for stream completion
	// The content is either already printed (raw mode) or will be printed on Close
	return nil
}

// Close implements StreamWriter
func (w *StdoutWriter) Close() error {
	if !w.pretty {
		// Add newlines at the end for raw mode
		_, err := fmt.Fprintln(os.Stdout)
		return err
	}
	
	// For pretty mode, render the full content with markdown
	if w.fullContent != "" {
		// Clean up common streaming artifacts
		cleanedContent := strings.TrimSpace(w.fullContent)
		if cleanedContent != "" {
			formatted, err := renderMarkdown(cleanedContent)
			if err != nil {
				// Fallback to plain text if rendering fails
				_, err = fmt.Fprintln(os.Stdout, cleanedContent)
				return err
			}
			_, err = fmt.Fprintln(os.Stdout, formatted)
			return err
		}
	}
	
	// If no content, just print a newline
	_, err := fmt.Fprintln(os.Stdout)
	return err
}

// renderMarkdown renders markdown content with glamour
func renderMarkdown(md string) (string, error) {
	out, err := glamour.Render(md, "dark")
	if err != nil {
		return "", err
	}
	return out, nil
}
