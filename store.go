package main

import (
	"fmt"
	"sync"
	"time"
)

type Record struct {
	content   []byte
	signature string
	createdAt int64
}

// Store Stores webhook payloads until they can be transferred to client
type Store interface {
	Put(requestId string, record Record)
	Get(requestId string) (Record, error)
	Await(requestId string) chan struct{}
	Delete(requestId string)
	GetOlderThan(time.Duration) []string
}

type InMemStore struct {
	store     sync.Map
	listeners sync.Map
}

func NewInMemStore() *InMemStore {
	return &InMemStore{
		store:     sync.Map{},
		listeners: sync.Map{},
	}
}

func (i *InMemStore) Put(requestId string, record Record) {
	i.store.Store(requestId, Record{record.content, record.signature, time.Now().Unix()})

	// Notify any listening clients
	if ch, ok := i.listeners.Load(requestId); ok {
		chx, _ := ch.(chan struct{})
		chx <- struct{}{}
	}
}

func (i *InMemStore) Get(requestId string) (Record, error) {
	if record, ok := i.store.Load(requestId); ok {
		return record.(Record), nil
	}

	return Record{}, fmt.Errorf("no response for request %s", requestId)
}

func (i *InMemStore) Await(requestId string) chan struct{} {
	ch := make(chan struct{})
	i.listeners.Store(requestId, ch)
	return ch
}

func (i *InMemStore) Delete(requestId string) {
	i.listeners.Delete(requestId)
	i.store.Delete(requestId)
}

func (i *InMemStore) GetOlderThan(duration time.Duration) []string {
	var requestsIds []string

	olderThanTimestamp := time.Now().Unix() - int64(duration.Seconds())
	i.store.Range(func(key, value interface{}) bool {
		if value.(Record).createdAt < olderThanTimestamp {
			requestsIds = append(requestsIds, key.(string))
		}
		return true
	})

	return requestsIds
}
