package stream

import (
	"fmt"
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
	
	// fmt.Printf("[TeeWriter] Writing chunk to %d writers (length: %d)\n", len(t.writers), len(chunk))
	
	// Write to all writers, collecting errors
	var firstErr error
	for i, writer := range t.writers {
		if err := writer.WriteChunk(chunk); err != nil {
			fmt.Printf("[TeeWriter] Writer %d failed: %v\n", i, err)
			if firstErr == nil {
				firstErr = err
			}
		} else {
			// fmt.Printf("[TeeWriter] Writer %d succeeded\n", i)
		}
	}
	return firstErr
}

// MarkStreamComplete marks the current stream as complete for all underlying writers
func (t *TeeWriter) MarkStreamComplete() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return nil
	}
	
	// Mark stream complete on all writers, collecting errors
	var firstErr error
	for i, writer := range t.writers {
		if err := writer.MarkStreamComplete(); err != nil {
			fmt.Printf("[TeeWriter] Writer %d MarkStreamComplete failed: %v\n", i, err)
			if firstErr == nil {
				firstErr = err
			}
		} else {
			// fmt.Printf("[TeeWriter] Writer %d MarkStreamComplete succeeded\n", i)
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
