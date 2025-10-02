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

	// Handle producer connections
	if role == "producer" {
		// Check authentication if token is required
		if *wsToken != "" {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Missing Authorization header"))
				return
			}

			// Check if it's a Bearer token
			expectedAuth := "Bearer " + *wsToken
			if authHeader != expectedAuth {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Invalid Authorization token"))
				return
			}
		}

		// Add producer to producers map
		clientsMu.Lock()
		producers[conn] = true
		clientsMu.Unlock()
		defer func() {
			clientsMu.Lock()
			delete(producers, conn)
			clientsMu.Unlock()
		}()

		// Handle messages from producer
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("producer read error: %v", err)
				break
			}

			// Broadcast message to all viewers
			clientsMu.RLock()
			for viewer := range viewers {
				err := viewer.WriteMessage(messageType, message)
				if err != nil {
					log.Printf("viewer write error: %v", err)
					// We'll remove the viewer later when we detect the error
				}
			}
			clientsMu.RUnlock()
		}
	} else {
		// Handle viewer connections
		clientsMu.Lock()
		viewers[conn] = true
		clientsMu.Unlock()
		defer func() {
			clientsMu.Lock()
			delete(viewers, conn)
			clientsMu.Unlock()
		}()

		// Keep the connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("viewer read error: %v", err)
				break
			}
		}
	}
}
