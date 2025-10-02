package stream

// StreamWriter defines the interface for writing streaming chunks
type StreamWriter interface {
	// WriteChunk writes a chunk of content to the stream
	WriteChunk(chunk string) error
	// Close closes the stream and releases any resources
	Close() error
}
