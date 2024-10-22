package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHandleClientStream_NoAuthHeader(t *testing.T) {
	// No auth header
	req, _ := http.NewRequest("GET", "/listen/asd", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleClientStream(context.Background()))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || !strings.Contains(rr.Body.String(), "unauthorized") {
		t.Errorf("handler returned wrong status code: got %v want %v (response body: %s)", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
}

func TestHandleClientStream_NoTokenGenerated(t *testing.T) {
	// No auth header
	req, _ := http.NewRequest("GET", "/listen/asd", nil)
	req.Header.Add("Authorization", "Bearer xxxxxx")
	streamsTokens = sync.Map{}
	s := strings.Builder{}
	log.SetOutput(&s)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleClientStream(context.Background()))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || !strings.Contains(rr.Body.String(), "unauthorized") {
		t.Errorf("handler returned wrong status code: got %v want %v (response body: %s)", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
	if !strings.Contains(s.String(), "no token found") {
		t.Errorf("expected no token log message, got %s", s.String())
	}
}

func TestHandleClientStream_InvalidTokens(t *testing.T) {
	// Prepare request
	req, _ := http.NewRequest("GET", "/listen/asd", nil)
	req.SetPathValue("request_id", "asd")
	req.Header.Add("Authorization", "Bearer xxxxxx")

	// Pre-fill streams tokens map
	streamsTokens = sync.Map{}
	streamsTokens.Store("asd", streamToken{"xxxxxx", time.Now().Add(-20 * time.Minute).Unix()})

	// Catch logs output
	s := strings.Builder{}
	log.SetOutput(&s)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleClientStream(context.Background()))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || !strings.Contains(rr.Body.String(), "unauthorized") {
		t.Errorf("handler returned wrong status code: got %v want %v (response body: %s)", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
	fmt.Println(s.String())
	if !strings.Contains(s.String(), "expired token") {
		t.Errorf("expected expired token log message, got %s", s.String())
	}
}

func TestHandleClientStream_InvalidToken(t *testing.T) {
	// Prepare request
	req, _ := http.NewRequest("GET", "/listen/asd", nil)
	req.SetPathValue("request_id", "asd")
	req.Header.Add("Authorization", "Bearer xxxxxx")

	// Pre-fill streams tokens map
	streamsTokens = sync.Map{}
	streamsTokens.Store("asd", streamToken{"yyyyyy", time.Now().Unix()})

	// Catch logs output
	s := strings.Builder{}
	log.SetOutput(&s)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleClientStream(context.Background()))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || !strings.Contains(rr.Body.String(), "unauthorized") {
		t.Errorf("handler returned wrong status code: got %v want %v (response body: %s)", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
	fmt.Println(s.String())
	if !strings.Contains(s.String(), "invalid token") {
		t.Errorf("expected invalid token log message, got %s", s.String())
	}
}

func TestHandleClientStream_WebhookResponseAlreadyPresent(t *testing.T) {
	// Prepare request
	req, _ := http.NewRequest("GET", "/listen/asd", nil)
	req.SetPathValue("request_id", "asd")
	req.Header.Add("Authorization", "Bearer a")

	// Pre-fill streams tokens map
	streamsTokens = sync.Map{}
	streamsTokens.Store("asd", streamToken{"a", time.Now().Unix()})

	store = NewInMemStore() // Initialize store
	store.Put("asd", Record{[]byte("content"), "signature", 0})

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleClientStream(context.Background()))

	handler.ServeHTTP(rr, req)
	expectedBody := "data: content\n\ndata: signature=signature\n\ndata: eot"
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), expectedBody) {
		t.Errorf("expected body 'data: server-gone', got %s", rr.Body.String())
	}

	// Confirm that token and store entry are deleted
	if _, err := store.Get("asd"); err == nil {
		t.Errorf("expected store entry to be gone but it's still present")
	}
	if _, ok := streamsTokens.Load("asd"); ok {
		t.Errorf("expected stream token to be gone but it's still present")
	}
}

func TestHandleClientStream_CloseStreamTimeout(t *testing.T) {
	// Prepare request
	req, _ := http.NewRequest("GET", "/listen/asd", nil)
	req.SetPathValue("request_id", "asd")
	req.Header.Add("Authorization", "Bearer a")

	// Pre-fill streams tokens map
	streamsTokens = sync.Map{}
	streamsTokens.Store("asd", streamToken{"a", time.Now().Unix()})

	store = NewInMemStore() // Initialize store
	requestTimeout = 0      // Set request timeout (seconds)

	// Catch logs
	s := strings.Builder{}
	log.SetOutput(&s)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleClientStream(context.Background()))

	handler.ServeHTTP(rr, req)
	fmt.Println(rr.Body.String())
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "data: server gone") {
		t.Errorf("expected body 'data: server gone', got %s", rr.Body.String())
	}
	if !strings.Contains(s.String(), "reason: timeout") {
		t.Errorf("expected log message 'reason: timeout', got %s", s.String())
	}
}

func TestHandleClientStream_ContextDone(t *testing.T) {
	// Prepare request
	req, _ := http.NewRequest("GET", "/listen/asd", nil)
	req.SetPathValue("request_id", "asd")
	req.Header.Add("Authorization", "Bearer a")

	// Pre-fill streams tokens map
	streamsTokens = sync.Map{}
	streamsTokens.Store("asd", streamToken{"a", time.Now().Unix()})

	store = NewInMemStore() // Initialize store
	requestTimeout = 10     // Set request timeout (seconds)

	// Catch logs
	s := strings.Builder{}
	log.SetOutput(&s)

	rr := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	handler := http.HandlerFunc(handleClientStream(ctx))

	// Cancel context in 100ms
	go func() {
		tc := time.NewTimer(100 * time.Millisecond)
		<-tc.C
		cancel()
	}()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "data: server gone") {
		t.Errorf("expected body 'data: server gone', got %s", rr.Body.String())
	}
	if !strings.Contains(s.String(), "reason: context done") {
		t.Errorf("expected log message 'reason: timeout', got %s", s.String())
	}
}

func TestHandleClientStream_RequestContextDone(t *testing.T) {
	// Prepare request
	req, _ := http.NewRequest("GET", "/listen/asd", nil)
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	req.SetPathValue("request_id", "asd")
	req.Header.Add("Authorization", "Bearer a")

	// Pre-fill streams tokens map
	streamsTokens = sync.Map{}
	streamsTokens.Store("asd", streamToken{"a", time.Now().Unix()})

	store = NewInMemStore() // Initialize store
	requestTimeout = 10     // Set request timeout (seconds)

	// Catch logs
	s := strings.Builder{}
	log.SetOutput(&s)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleClientStream(context.Background()))

	// Cancel context in 100ms
	go func() {
		tc := time.NewTimer(100 * time.Millisecond)
		<-tc.C
		cancel()
	}()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected response code %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(s.String(), "client asd disconnected") {
		t.Errorf("expected log message 'client asd disconnected', got %s", s.String())
	}
}

func TestHandleClientStream_SendResponse(t *testing.T) {
	// Prepare request
	req, _ := http.NewRequest("GET", "/listen/asd", nil)
	req.SetPathValue("request_id", "asd")
	req.Header.Add("Authorization", "Bearer a")

	// Pre-fill streams tokens map
	streamsTokens = sync.Map{}
	streamsTokens.Store("asd", streamToken{"a", time.Now().Unix()})

	store = NewInMemStore() // Initialize store
	requestTimeout = 10     // Set request timeout (seconds)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleClientStream(context.Background()))

	// Send response after 100ms
	go func() {
		tc := time.NewTimer(100 * time.Millisecond)
		<-tc.C
		store.Put("asd", Record{[]byte("content"), "signature", 0})
	}()

	handler.ServeHTTP(rr, req)
	expectedBody := "data: content\n\ndata: signature=signature\n\ndata: eot"
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), expectedBody) {
		t.Errorf("expected body '%s', got %s", expectedBody, rr.Body.String())
	}

	// Confirm that token and store entry are deleted
	if _, err := store.Get("asd"); err == nil {
		t.Errorf("expected store entry to be gone but it's still present")
	}
	if _, ok := streamsTokens.Load("asd"); ok {
		t.Errorf("expected stream token to be gone but it's still present")
	}
}
