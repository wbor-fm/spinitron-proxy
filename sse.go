package main

import (
	"log"
	"net/http"
	"sync"
)

var (
	sseClients  []chan string
	sseClientsM sync.Mutex // to synchronize access to sseClients
)

// spinEventsHandler is an HTTP handler that streams server-sent events (SSE) to
// clients. It creates a channel for each client to receive messages and adds
// the channel to the sseClients array. It then sends messages to the client
// when they are available.
func spinEventsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the client supports server-sent events via the http.Flusher
	// interface. If it doesn't, return an error.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create a channel for this client to receive messages
	msgChan := make(chan string, 1)

	// Lock, modify slice, unlock
	sseClientsM.Lock()
	sseClients = append(sseClients, msgChan)
	log.Println("sse.connect", len(sseClients))
	sseClientsM.Unlock()

	// Clean up when the client disconnects: remove the channel from the array
	// and close the channel.
	defer func() {
		sseClientsM.Lock()
		defer sseClientsM.Unlock()
		for i, c := range sseClients {
			if c == msgChan {
				// Slice the client out of the array
				sseClients = append(sseClients[:i], sseClients[i+1:]...)
				log.Println("sse.disconnect", len(sseClients))
				break
			}
		}
		close(msgChan)
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-r.Context().Done():
			// Client closed connection
			return
		case msg := <-msgChan:
			// Send SSE message
			_, _ = w.Write([]byte(msg + "\n\n"))
			flusher.Flush()
		}
	}
}

// Send `msg` to all SSE clients
func BroadcastSpinMessage(msg string) {
	sseClientsM.Lock()
	defer sseClientsM.Unlock()
	for _, c := range sseClients {
		c <- "data: " + msg
	}
}
