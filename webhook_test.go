package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleIncomingWebhook_NoSignatureHeader(t *testing.T) {
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBufferString(""))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleIncomingWebhook)

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "bad request") {
		t.Errorf("handler returned wrong status code: got %v want %v (response body: %s)", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestHandleIncomingWebhook_InvalidBody(t *testing.T) {
	bodies := []string{"", "asdfasdfasdf", `{"json": "but incorrect fields"}`}

	for _, body := range bodies {
		req, _ := http.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
		req.Header.Set("X-BASETEN-SIGNATURE", "asd")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleIncomingWebhook)

		s := strings.Builder{}
		log.SetOutput(&s)
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "bad request") {
			t.Errorf("handler returned wrong status code: got %v want %v (body: %s)", rr.Code, http.StatusBadRequest, rr.Body.String())
		}

		if !strings.Contains(body, "json") && !strings.Contains(s.String(), "failed to unmarshal json body") {
			t.Errorf("expected unmarshal error to be reported, got: %s", s.String())
		}
	}
}

func TestHandleIncomingWebhook_Valid(t *testing.T) {
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBufferString(`{"request_id": "asd"}`))
	req.Header.Set("X-BASETEN-SIGNATURE", "xxx")

	store = NewInMemStore()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleIncomingWebhook)

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v (response body: %s)", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Assert record inserted to the store
	record, err := store.Get("asd")
	if err != nil {
		t.Errorf("expected request stored, got err: %v", err)
	}

	if record.signature != "xxx" || string(record.content) != `{"request_id": "asd"}` {
		t.Errorf(
			"expected stored request to be signature=%s and content=%s, got signature=%s and body=%s",
			"xxx", `{"request_id": "asd"}`, record.signature, record.content,
		)
	}
}
