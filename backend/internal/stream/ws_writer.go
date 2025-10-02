package stream

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage represents the JSON message structure sent over WebSocket
type WSMessage struct {
	T     int64  `json:"t"`   // Unix timestamp in milliseconds
	Seq   int64  `json:"seq"` // Monotonically increasing sequence number
	Chunk string `json:"chunk"`
}

// WSWriter implements StreamWriter for writing to a WebSocket
type WSWriter struct {
	url       string
	token     string
	conn      *websocket.Conn
	mu        sync.Mutex
	seq       int64
	closed    int32 // atomic flag
	buffer    []WSMessage
	bufferMu  sync.Mutex
	bufferMax int
	dialer    *websocket.Dialer
}

// NewWSWriter creates a new WSWriter
func NewWSWriter(url, token string) *WSWriter {
	return &WSWriter{
		url:       url,
		token:     token,
		buffer:    make([]WSMessage, 0),
		bufferMax: 100, // Max buffer size
		dialer:    &websocket.Dialer{},
	}
}

// WriteChunk writes a chunk to the WebSocket
func (w *WSWriter) WriteChunk(chunk string) error {
	if atomic.LoadInt32(&w.closed) == 1 {
		return fmt.Errorf("writer is closed")
	}

	// Create message with timestamp and sequence number
	msg := WSMessage{
		T:     time.Now().UnixMilli(),
		Seq:   atomic.AddInt64(&w.seq, 1),
		Chunk: chunk,
	}

	// Try to send immediately if connected
	w.mu.Lock()
	if w.conn != nil {
		err := w.conn.WriteJSON(msg)
		w.mu.Unlock()
		if err == nil {
			return nil
		}
		// If there was an error, close the connection and fall through to buffering
		w.conn.Close()
		w.conn = nil
	} else {
		w.mu.Unlock()
	}

	// Buffer the message for later sending
	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()

	// Apply backpressure protection - drop oldest messages if buffer is full
	if len(w.buffer) >= w.bufferMax {
		// Remove oldest message (at index 0)
		w.buffer = w.buffer[1:]
	}

	w.buffer = append(w.buffer, msg)
	return nil
}

// Close closes the WebSocket connection
func (w *WSWriter) Close() error {
	atomic.StoreInt32(&w.closed, 1)
	
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// connect establishes a WebSocket connection with optional authentication
func (w *WSWriter) connect() error {
	header := http.Header{}
	if w.token != "" {
		header.Set("Authorization", "Bearer "+w.token)
	}

	conn, _, err := w.dialer.Dial(w.url, header)
	if err != nil {
		return err
	}

	w.conn = conn
	return nil
}

// flushBuffer sends all buffered messages
func (w *WSWriter) flushBuffer() {
	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()

	if len(w.buffer) == 0 {
		return
	}

	// Send all buffered messages
	for _, msg := range w.buffer {
		err := w.conn.WriteJSON(msg)
		if err != nil {
			// If we fail to send, put messages back in buffer and return
			// In a real implementation, you might want to handle this differently
			return
		}
	}

	// Clear buffer after successful send
	w.buffer = w.buffer[:0]
}

// StartReconnectLoop starts the reconnection loop in a goroutine
func (w *WSWriter) StartReconnectLoop() {
	go func() {
		backoff := time.Second
		maxBackoff := 32 * time.Second

		for atomic.LoadInt32(&w.closed) == 0 {
			w.mu.Lock()
			connected := w.conn != nil
			w.mu.Unlock()

			if !connected {
				err := w.connect()
				if err == nil {
					// Successfully reconnected, flush buffer
					w.flushBuffer()
					// Reset backoff on successful connection
					backoff = time.Second
				} else {
					// Exponential backoff with jitter
					jitter := time.Duration(time.Now().UnixNano() % int64(backoff))
					sleepTime := backoff + jitter/2
					
					time.Sleep(sleepTime)
					
					// Increase backoff for next attempt
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}
			} else {
				// We're connected, check if connection is still alive with ping
				w.mu.Lock()
				conn := w.conn
				w.mu.Unlock()
				
				if conn != nil {
					// Send ping every 20 seconds
					err := conn.WriteMessage(websocket.PingMessage, []byte{})
					if err != nil {
						// Connection is dead, close it and let it reconnect
						w.mu.Lock()
						if w.conn != nil {
							w.conn.Close()
							w.conn = nil
						}
						w.mu.Unlock()
					}
				}
				
				time.Sleep(20 * time.Second)
			}
		}
	}()
}
