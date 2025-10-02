package stream

import (
	"sync"
)

// TeeWriter implements StreamWriter by writing to multiple StreamWriter instances
type TeeWriter struct {
	writers []StreamWriter
	mu      sync.Mutex
	closed  bool
}

// NewTeeWriter creates a new TeeWriter that writes to all provided writers
func NewTeeWriter(writers ...StreamWriter) *TeeWriter {
	return &TeeWriter{
		writers: writers,
	}
}

// WriteChunk writes a chunk to all underlying writers
func (t *TeeWriter) WriteChunk(chunk string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return nil // Don't write to closed writers
	}
	
	// Write to all writers, collecting errors
	var firstErr error
	for _, writer := range t.writers {
		if err := writer.WriteChunk(chunk); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Close closes all underlying writers
func (t *TeeWriter) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return nil
	}
	
	t.closed = true
	
	// Close all writers, collecting errors
	var firstErr error
	for _, writer := range t.writers {
		if err := writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
