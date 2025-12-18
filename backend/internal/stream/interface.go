package stream

// StreamWriter defines the interface for writing streaming chunks
type StreamWriter interface {
	// WriteChunk writes a chunk of content to the stream
	WriteChunk(chunk string) error
	// MarkStreamComplete marks the current stream as complete without closing the connection
	MarkStreamComplete() error
	// Close closes the stream and releases any resources (should only be called on terminal shutdown)
	Close() error
}
