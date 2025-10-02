package main

import (
	"flag"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", ":4000", "http service address")
	wsToken  = flag.String("ws-token", "", "Authorization token for producer connections")
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin
		},
	}
	producers = make(map[*websocket.Conn]bool)
	viewers   = make(map[*websocket.Conn]bool)
	clientsMu sync.RWMutex
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	http.HandleFunc("/stream", handleStream)
	log.Printf("WebSocket relay server starting on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	// Check role parameter
	role := r.URL.Query().Get("role")
	if role != "producer" && role != "viewer" {
		http.Error(w, "Invalid role parameter", http.StatusBadRequest)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer conn.Close()

	log.Printf("[%s] New connection from %s", role, r.RemoteAddr)

	// Handle producer connections
	if role == "producer" {
		// Check authentication if token is required
		if *wsToken != "" {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Missing Authorization header"))
				log.Printf("[producer] Missing Authorization header from %s", r.RemoteAddr)
				return
			}

			// Check if it's a Bearer token
			expectedAuth := "Bearer " + *wsToken
			if authHeader != expectedAuth {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Invalid Authorization token"))
				log.Printf("[producer] Invalid Authorization token from %s", r.RemoteAddr)
				return
			}
		}

		log.Printf("[producer] Producer connected from %s", r.RemoteAddr)

		// Add producer to producers map
		clientsMu.Lock()
		producers[conn] = true
		clientsMu.Unlock()
		defer func() {
			clientsMu.Lock()
			delete(producers, conn)
			clientsMu.Unlock()
			log.Printf("[producer] Producer disconnected from %s", r.RemoteAddr)
		}()

		// Handle messages from producer
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[producer] Producer read error from %s: %v", r.RemoteAddr, err)
				break
			}

			// Log incoming message from producer
			if messageType == websocket.TextMessage {
				// log.Printf("[producer] Message received from %s: %s", r.RemoteAddr, string(message))
			} else {
				// log.Printf("[producer] Binary message received from %s (size: %d bytes)", r.RemoteAddr, len(message))
			}

			// Broadcast message to all viewers
			clientsMu.RLock()
			// viewerCount := len(viewers)
			// if viewerCount > 0 {
			// 	log.Printf("[producer] Broadcasting to %d viewers", viewerCount)
			// }
			for viewer := range viewers {
				err := viewer.WriteMessage(messageType, message)
				if err != nil {
					log.Printf("[producer] Viewer write error: %v", err)
					// We'll remove the viewer later when we detect the error
				}
			}
			clientsMu.RUnlock()
		}
	} else {
		// Handle viewer connections
		clientsMu.Lock()
		viewers[conn] = true
		viewerCount := len(viewers)
		clientsMu.Unlock()
		log.Printf("[viewer] Viewer connected from %s (total viewers: %d)", r.RemoteAddr, viewerCount)

		defer func() {
			clientsMu.Lock()
			delete(viewers, conn)
			viewerCount := len(viewers)
			clientsMu.Unlock()
			log.Printf("[viewer] Viewer disconnected from %s (total viewers: %d)", r.RemoteAddr, viewerCount)
		}()

		// Keep the connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[viewer] Viewer read error from %s: %v", r.RemoteAddr, err)
				break
			}
		}
	}
}
