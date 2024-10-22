package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// handleIncomingWebhook validates and stores webhook payloads received from Baseten to be forwarded to the client
func handleIncomingWebhook(w http.ResponseWriter, r *http.Request) {
	// Drop requests without signature header
	signature := r.Header.Get("X-BASETEN-SIGNATURE")
	if signature == "" {
		log.Println("webhook request received with no signature, dropping")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Printf("failed to read body: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	decoded := struct {
		RequestId string `json:"request_id"`
	}{}

	if err = json.Unmarshal(b, &decoded); err != nil {
		log.Printf("failed to unmarshal json body: %s\n", string(b))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if decoded.RequestId == "" {
		log.Printf("webhook delivered but missing request_id: %s\n", string(b))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	log.Printf("received webhook request with id=%s\n", decoded.RequestId)
	promWebhooksReceived.Inc()

	store.Put(decoded.RequestId, Record{content: b, signature: signature})
}
