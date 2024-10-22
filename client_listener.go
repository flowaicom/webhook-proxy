package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func handleClientStream(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestId := r.PathValue("request_id")
		log.Printf("new listener, request_id: %s\n", requestId)

		// Auth
		if ok := authClientStream(w, r, requestId); !ok {
			return
		}

		// Create stream
		flusher, ok := w.(http.Flusher)
		if !ok {
			log.Printf("failed to create stream, request_id: %s\n", requestId)
			http.Error(w, "failed to open stream, try again later", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		clientListenLoop(w, r, requestId, flusher, ctx)
	}
}

// authClientStream checks provided Bearer token and validates it with the expected (previously generated) stream token
func authClientStream(w http.ResponseWriter, r *http.Request, requestId string) bool {
	// Auth
	providedToken := r.Header.Get("Authorization")
	if providedToken == "" {
		log.Printf("client connected without authorization header: %s\n", requestId)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}

	t, ok := streamsTokens.Load(requestId)
	if !ok {
		log.Printf("client connected but no token found for request_id: %s\n", requestId)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}

	requiredToken := t.(streamToken)

	if requiredToken.token != strings.TrimPrefix(providedToken, "Bearer ") {
		log.Printf("client provided invalid token for request_id: %s\n", requestId)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}

	if requiredToken.expiresAt < time.Now().Unix() {
		log.Printf("client provided expired token for request_id: %s\n", requestId)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}

	return true
}

// clientListenLoop holds user http stream connection, streams response when webhook response is available
func clientListenLoop(w http.ResponseWriter, r *http.Request, requestId string, flusher http.Flusher, ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	timeout := time.NewTimer(time.Duration(requestTimeout) * time.Second)
	defer ticker.Stop()
	defer timeout.Stop()

	// Instrument
	promOpenClientConnections.Inc()
	promTotalClientConnections.Inc()
	defer promOpenClientConnections.Dec()

	// Check if request payload is already there and awaiting
	_, err := store.Get(requestId)
	if err == nil {
		sendClientResponse(w, requestId, flusher)
		return
	}

	for {
		select {
		case <-r.Context().Done():
			log.Printf("client %s disconnected\n", requestId)
			return
		case <-store.Await(requestId):
			sendClientResponse(w, requestId, flusher)
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(w, "data: keep-alive\n\n"); err != nil {
				log.Printf("failed to ping client (request_id: %s): %v\n", requestId, err)
				return
			}
			flusher.Flush()
		case <-timeout.C:
			closeClientConnection(w, requestId, flusher, "timeout")
			promTimedOutClients.Inc()
			return
		case <-ctx.Done():
			closeClientConnection(w, requestId, flusher, "context done")
			return
		}
	}
}

// sendClientResponse responds to client with the actual webhook payload when it's received
func sendClientResponse(w http.ResponseWriter, requestId string, flusher http.Flusher) {
	log.Printf("responding to request %s\n", requestId)
	record, err := store.Get(requestId)
	if err != nil {
		log.Printf("failed to retrieve response for (request_id: %s): %v\n", requestId, err)
		http.Error(w, "failed to retrieve response", http.StatusInternalServerError)
		return
	}
	if _, err = fmt.Fprintf(w, "data: %s\n\ndata: signature=%s\n\n", record.content, record.signature); err != nil {
		log.Printf("failed to write response (request_id: %s): %v\n", requestId, err)
		return
	}
	flusher.Flush()
	if _, err = fmt.Fprintf(w, "data: eot\n\n"); err != nil {
		log.Printf("failed to write end of transmision response (request_id: %s): %v\n", requestId, err)
		return
	}
	flusher.Flush()

	// Cleanup
	store.Delete(requestId)
	streamsTokens.Delete(requestId)
	promActiveTokens.Dec()

	return
}

func closeClientConnection(w http.ResponseWriter, requestId string, flusher http.Flusher, reason string) {
	log.Printf("closing client connection (request_id: %s, reason: %s)", requestId, reason)
	if _, err := fmt.Fprintf(w, "data: server gone\n\n"); err != nil {
		log.Printf("failed to notify client about shutdown (request_id: %s): %v\n", requestId, err)
		return
	}
	flusher.Flush()
	return
}
