package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var streamTokenExpiration = 15 * time.Minute

// streamToken holds the token required for connecting to /listen endpoint and it's expiration
type streamToken struct {
	token     string
	expiresAt int64
}

var streamsTokens sync.Map // map[requestId string]streamToken -

// handleCreateToken handles `POST /token` route. Accepts `request_id` field in JSON body, generates and stores token
// for accessing the stream for that request_id.
func handleCreateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestId string `json:"request_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.RequestId == "" {
		log.Printf("error decoding create stream token request or request id is empty: %v", err)
		http.Error(w, "Bad request. Field `request_id` (string) is required.", http.StatusBadRequest)
		return
	}

	token, err := generateSecureToken(16)
	if err != nil {
		log.Printf("error generating token (request_id: %s): %v", req.RequestId, err)
		http.Error(w, "cannot generate token", http.StatusInternalServerError)
		return
	}

	if _, ok := streamsTokens.Load(req.RequestId); ok {
		log.Printf("token already exists (request_id: %s)", req.RequestId)
		http.Error(w, "token already exists", http.StatusConflict)
		return
	}

	expiresAt := time.Now().Add(streamTokenExpiration).Unix()
	streamsTokens.Store(req.RequestId, streamToken{token, expiresAt})
	promActiveTokens.Inc()
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{"token": token, "expires_at": strconv.FormatInt(expiresAt, 10)})
	if err != nil {
		log.Printf("error responding with token (request_id: %s): %v", err, req.RequestId)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
