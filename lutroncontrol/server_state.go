package main

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/unixpickle/essentials"
	"github.com/unixpickle/lutronbroker/lutronbroker"
)

// Cache is an interface for a KV store that is implemented by ServerState.
type Cache interface {
	GetCache(key string) (any, bool)
	SetCache(key string, obj any)
}

// ServerState is the state (e.g. session info) that the server saves across
// runs.
//
// Methods are safe to call concurrently from multiple Goroutines.
type ServerState struct {
	lock         sync.Mutex
	brokerCreds  *lutronbroker.BrokerCredentials
	cache        map[string]any
	cacheIsSaved bool
}

// NewServerState creates or loads the state from a file.
func NewServerState(path string) (state *ServerState, err error) {
	defer essentials.AddCtxTo("load server state", &err)

	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return &ServerState{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ServerState{}, nil
		}
		return nil, err
	}
	var obj struct {
		BrokerCreds *lutronbroker.BrokerCredentials
		Cache       map[string]any
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	if obj.Cache == nil {
		obj.Cache = map[string]any{}
	}
	return &ServerState{brokerCreds: obj.BrokerCreds, cache: obj.Cache, cacheIsSaved: true}, nil
}

func (s *ServerState) BrokerCreds() *lutronbroker.BrokerCredentials {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.brokerCreds
}

func (s *ServerState) SetBrokerCreds(b *lutronbroker.BrokerCredentials) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.brokerCreds = b
}

// GetCache gets an object stored under the given key.
func (s *ServerState) GetCache(key string) (any, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	obj, ok := s.cache[key]
	return obj, ok
}

// SetCache updates an object stored under the given key.
// This marks the cache as unsaved, as indicated by
// CacheIsSaved(), until a Save() call succeeds.
func (s *ServerState) SetCache(key string, obj any) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.cache[key] = obj
	s.cacheIsSaved = false
}

// CacheIsSaved indicates whether or not the latest cache
// has been saved to disk.
func (s *ServerState) CacheIsSaved() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.cacheIsSaved
}

// Save writes the state to a file.
func (s *ServerState) Save(path string) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	defer essentials.AddCtxTo("load server state", &err)

	var obj struct {
		BrokerCreds *lutronbroker.BrokerCredentials
		Cache       map[string]any
	}
	obj.BrokerCreds = s.brokerCreds
	obj.Cache = s.cache

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0600)
	if err == nil {
		s.cacheIsSaved = true
	}
	return err
}
