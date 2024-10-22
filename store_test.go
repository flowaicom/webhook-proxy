package main

// Test InMem store

import (
	"testing"
	"time"
)

func TestInMemStorePutAndGet(t *testing.T) {
	store := NewInMemStore()
	requestId := "request1"
	response := []byte("response1")

	store.Put(requestId, Record{content: response})

	got, err := store.Get(requestId)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(got.content) != string(response) {
		t.Fatalf("expected %s, got %s", response, got.content)
	}
}

func TestInMemStoreGetNonExistent(t *testing.T) {
	store := NewInMemStore()
	_, err := store.Get("non_existent_request")
	if err == nil {
		t.Fatal("expected error for non-existent request")
	}
}

func TestInMemStoreAwait(t *testing.T) {
	store := NewInMemStore()
	awaitChannel := store.Await("request2")

	go func() {
		time.Sleep(time.Millisecond * 500) // Simulating delay
		store.Put("request2", Record{content: []byte("response2")})
	}()

	select {
	case <-awaitChannel: // Successfully received signal
		// No action needed
	case <-time.After(time.Second):
		t.Fatal("await timed out")
	}

	got, err := store.Get("request2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(got.content) != "response2" {
		t.Fatalf("expected 'response2', got %s", got.content)
	}
}

func TestInMemStoreDelete(t *testing.T) {
	store := NewInMemStore()
	requestId := "request1"
	response := []byte("response1")

	store.Put(requestId, Record{content: response})
	_, err := store.Get("request1")
	if err != nil {
		t.Fatal("put request failed before deleting")
	}
	store.Delete(requestId)

	_, err = store.Get(requestId)
	if err == nil {
		t.Fatal("expected error when getting after delete")
	}
}

func TestInMemStoreGetOlderThan(t *testing.T) {
	store := NewInMemStore()

	store.store.Store("request1", Record{[]byte("response1"), "", time.Now().Unix() - 15})
	store.store.Store("request2", Record{[]byte("response2"), "", time.Now().Unix() - 10})
	store.store.Store("request3", Record{[]byte("response3"), "", time.Now().Unix() - 5})
	store.store.Store("request4", Record{[]byte("response4"), "", time.Now().Unix()})

	keys := store.GetOlderThan(time.Second * 5)
	if len(keys) != 2 {
		t.Fatalf("expected 2 requests ids to be returned")
	}
	if (keys[0] != "request1" && keys[0] != "request2") && (keys[1] != "request2" && keys[1] != "request1") {
		t.Fatalf("returned requestIds expected to be request1 and request2, got %s and %s", keys[0], keys[1])
	}
}
