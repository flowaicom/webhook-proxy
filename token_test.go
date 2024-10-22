package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleCreateToken(t *testing.T) {
	tests := []struct {
		body         string
		expectedCode int
	}{
		{`{"request_id": "req1"}`, http.StatusOK},
		{`{"request_id": ""}`, http.StatusBadRequest},
	}

	for _, test := range tests {
		req, _ := http.NewRequest("POST", "/token", bytes.NewBuffer([]byte(test.body)))

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleCreateToken)

		handler.ServeHTTP(rr, req)

		if rr.Code != test.expectedCode {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, test.expectedCode)
		}
	}
}

func TestHandleCreateToken_TokenAlreadyExists(t *testing.T) {
	streamsTokens.Store("req1", streamToken{})
	req, _ := http.NewRequest("POST", "/token", bytes.NewBufferString(`{"request_id":"req1"}`))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleCreateToken)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusConflict)
	}
}
