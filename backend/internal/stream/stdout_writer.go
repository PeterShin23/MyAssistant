package stream

import (
	"fmt"
	"os"
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

// Close implements StreamWriter
func (w *StdoutWriter) Close() error {
	if !w.pretty {
		// Add newlines at the end for raw mode
		_, err := fmt.Fprintln(os.Stdout)
		return err
	}
	
	// For pretty mode, we should render the full content
	// However, since we don't have access to the glamour package here,
	// we'll just print the accumulated content
	// The actual pretty rendering is done in the processor
	_, err := fmt.Fprintln(os.Stdout, w.fullContent)
	return err
}
