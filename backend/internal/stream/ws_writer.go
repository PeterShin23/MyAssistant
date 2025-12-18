package stream

import (
	"fmt"
	"net/http"
	"strings"
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

// CommandHandler is a callback function for handling commands received via WebSocket
type CommandHandler func(command string)

// WSWriter implements StreamWriter for writing to a WebSocket
type WSWriter struct {
	url            string
	token          string
	conn           *websocket.Conn
	mu             sync.Mutex
	seq            int64
	closed         int32 // atomic flag
	buffer         []WSMessage
	bufferMu       sync.Mutex
	bufferMax      int
	dialer         *websocket.Dialer
	commandHandler CommandHandler
	readLoopDone   chan struct{}
}

// NewWSWriter creates a new WSWriter and establishes initial connection
func NewWSWriter(url, token string) *WSWriter {
	w := &WSWriter{
		url:          url,
		token:        token,
		buffer:       make([]WSMessage, 0),
		bufferMax:    100, // Max buffer size
		dialer:       &websocket.Dialer{},
		readLoopDone: make(chan struct{}),
	}

	// Establish initial connection immediately
	if err := w.connect(); err != nil {
		fmt.Printf("Warning: initial WebSocket connection failed: %v\n", err)
		// Start the reconnection loop in the background
		go w.StartReconnectLoop()
	} else {
		fmt.Printf("WebSocket connection established to %s\n", url)
	}

	return w
}

// SetCommandHandler sets the callback function for handling commands
func (w *WSWriter) SetCommandHandler(handler CommandHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.commandHandler = handler
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
		if err != nil {
			fmt.Printf("[WSWriter] Failed to send message (seq=%d): %v\n", msg.Seq, err)
			// For broken pipe errors, close the connection so reconnect loop can handle it
			if strings.Contains(err.Error(), "broken pipe") || 
			   strings.Contains(err.Error(), "connection reset") ||
			   strings.Contains(err.Error(), "write: ") {
				fmt.Printf("[WSWriter] Connection error, closing for reconnect (seq=%d)\n", msg.Seq)
				w.conn.Close()
				w.conn = nil
			}
		} 
		// else {
		// 	fmt.Printf("[WSWriter] Sent message (seq=%d, chunklen=%d): %q\n", msg.Seq, len(msg.Chunk), msg.Chunk)
		// }
		w.mu.Unlock()
	} else {
		w.mu.Unlock()
		// fmt.Printf("[WSWriter] No connection, buffering message (seq=%d)\n", msg.Seq)
	}

	// Buffer the message for later sending
	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()

	// Apply backpressure protection - drop oldest messages if buffer is full
	if len(w.buffer) >= w.bufferMax {
		// Remove oldest message (at index 0)
		// dropped := w.buffer[0]
		w.buffer = w.buffer[1:]
		// fmt.Printf("[WSWriter] Buffer full, dropped oldest message (seq=%d)\n", dropped.Seq)
	}

	w.buffer = append(w.buffer, msg)
	return nil
}

// Close closes the WebSocket connection and clears the buffer
// This should only be called when the terminal is shutting down
func (w *WSWriter) Close() error {
	atomic.StoreInt32(&w.closed, 1)
	
	// Clear the buffer when closing
	w.ClearBuffer()
	
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.conn != nil {
		fmt.Printf("[WSWriter] Closing WebSocket connection (terminal shutdown)\n")
		return w.conn.Close()
	}
	return nil
}

// MarkStreamComplete marks the current stream as complete without closing the connection
func (w *WSWriter) MarkStreamComplete() error {
	// fmt.Printf("[WSWriter] Stream completed, keeping connection open\n")
	
	// Clear the buffer but keep the connection alive
	w.ClearBuffer()
	
	// Don't close the connection - just mark that we're done with this stream
	// The connection will remain open for the next request
	return nil
}

// IsConnected returns true if the WebSocket connection is active
func (w *WSWriter) IsConnected() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn != nil
}

// ClearBuffer clears all buffered messages
func (w *WSWriter) ClearBuffer() {
	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()
	
	// bufferSize := len(w.buffer)
	w.buffer = w.buffer[:0]
	// fmt.Printf("[WSWriter] Buffer cleared, removed %d messages\n", bufferSize)
}

// ForceReconnection closes the current connection and forces an immediate reconnection
func (w *WSWriter) ForceReconnection() {
	fmt.Printf("[WSWriter] Force reconnection requested\n")
	
	w.mu.Lock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.mu.Unlock()
	
	// The reconnect loop will detect the nil connection and reconnect
}

// connect establishes a WebSocket connection with optional authentication
func (w *WSWriter) connect() error {
	header := http.Header{}
	if w.token != "" {
		header.Set("Authorization", "Bearer "+w.token)
	}

	// Ensure the URL has the role parameter
	url := w.url
	if !strings.Contains(url, "?role=") && !strings.Contains(url, "&role=") {
		if strings.Contains(url, "?") {
			url += "&role=producer"
		} else {
			url += "?role=producer"
		}
	}

	conn, _, err := w.dialer.Dial(url, header)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", url, err)
	}

	w.conn = conn

	// Start reading messages from the WebSocket
	w.startReadLoop()

	return nil
}

// startReadLoop starts a goroutine to read incoming messages from the WebSocket
func (w *WSWriter) startReadLoop() {
	go func() {
		for {
			// Check if connection is closed
			w.mu.Lock()
			conn := w.conn
			w.mu.Unlock()

			if conn == nil {
				return // Connection closed, exit read loop
			}

			// Read message from WebSocket
			var msg map[string]interface{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				// Connection error, exit read loop
				// The reconnect loop will handle reconnection
				fmt.Printf("[WSWriter] Read error: %v\n", err)
				return
			}

			// Check if this is a command message
			if msgType, ok := msg["type"].(string); ok && msgType == "command" {
				if command, ok := msg["command"].(string); ok {
					fmt.Printf("[WSWriter] Received command: %s\n", command)

					// Call command handler if set
					w.mu.Lock()
					handler := w.commandHandler
					w.mu.Unlock()

					if handler != nil {
						// Execute handler in a goroutine to avoid blocking read loop
						go handler(command)
					}
				}
			}
		}
	}()
}

// flushBuffer sends all buffered messages
func (w *WSWriter) flushBuffer() {
	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()

	if len(w.buffer) == 0 {
		return
	}

	// fmt.Printf("[WSWriter] Flushing %d buffered messages\n", len(w.buffer))

	// Create a copy of the buffer to iterate over
	bufferCopy := make([]WSMessage, len(w.buffer))
	copy(bufferCopy, w.buffer)

	// Send all buffered messages
	successCount := 0
	for i, msg := range bufferCopy {
		err := w.conn.WriteJSON(msg)
		if err != nil {
			fmt.Printf("[WSWriter] Failed to send buffered message %d (seq=%d): %v\n", i, msg.Seq, err)
			// If we fail to send, close the connection to trigger reconnect
			if strings.Contains(err.Error(), "broken pipe") || 
			   strings.Contains(err.Error(), "connection reset") ||
			   strings.Contains(err.Error(), "write: ") {
				fmt.Printf("[WSWriter] Connection error during flush, closing for reconnect\n")
				w.mu.Lock()
				if w.conn != nil {
					w.conn.Close()
					w.conn = nil
				}
				w.mu.Unlock()
			}
			// Stop sending more messages, connection might be dead
			break
		}
		// fmt.Printf("[WSWriter] Sent buffered message %d (seq=%d)\n", i, msg.Seq)
		successCount++
	}

	// Only clear the messages that were successfully sent
	if successCount > 0 {
		w.buffer = w.buffer[successCount:]
		// fmt.Printf("[WSWriter] Successfully flushed %d messages, %d remaining\n", successCount, len(w.buffer))
	} else {
		// fmt.Printf("[WSWriter] No messages were flushed, keeping all %d messages\n", len(w.buffer))
	}
}

// StartReconnectLoop starts the reconnection loop in a goroutine
func (w *WSWriter) StartReconnectLoop() {
	go func() {
		backoff := time.Second
		maxBackoff := 32 * time.Second

		for atomic.LoadInt32(&w.closed) == 0 {
			w.mu.Lock()
			conn := w.conn
			w.mu.Unlock()

			if conn == nil {
				fmt.Printf("[WSWriter] Connection is nil, attempting to reconnect...\n")
				err := w.connect()
				if err == nil {
					// Successfully reconnected, flush buffer
					fmt.Printf("[WSWriter] Reconnected successfully, flushing buffer\n")
					w.flushBuffer()
					// Reset backoff on successful connection
					backoff = time.Second
				} else {
					// Exponential backoff with jitter
					jitter := time.Duration(time.Now().UnixNano() % int64(backoff))
					sleepTime := backoff + jitter/2
					
					fmt.Printf("[WSWriter] Reconnection failed (%v), retrying in %v\n", err, sleepTime)
					time.Sleep(sleepTime)
					
					// Increase backoff for next attempt
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}
			} else {
				// We're connected, check if connection is still alive with ping
				err := conn.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					fmt.Printf("[WSWriter] Ping failed, connection lost: %v\n", err)
					// Connection is dead, close it and let it reconnect
					w.mu.Lock()
					if w.conn != nil {
						w.conn.Close()
						w.conn = nil
					}
					w.mu.Unlock()
				} else {
					// Connection is healthy, sleep before next ping
					time.Sleep(20 * time.Second)
				}
			}
		}
	}()
}
