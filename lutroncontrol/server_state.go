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
	GetCache(key string, out any) bool
	SetCache(key string, obj any)
	ClearCache()
}

// ServerState is the state (e.g. session info) that the server saves across
// runs.
//
// Methods are safe to call concurrently from multiple Goroutines.
type ServerState struct {
	lock         sync.Mutex
	brokerCreds  *lutronbroker.BrokerCredentials
	cache        map[string]json.RawMessage
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
		Cache       map[string]json.RawMessage
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	if obj.Cache == nil {
		obj.Cache = map[string]json.RawMessage{}
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
func (s *ServerState) GetCache(key string, out any) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	if obj, ok := s.cache[key]; !ok {
		return false
	} else {
		if err := json.Unmarshal(obj, out); err != nil {
			panic("get cache error: " + err.Error())
		}
		return true
	}
}

// SetCache updates an object stored under the given key.
// This marks the cache as unsaved, as indicated by
// CacheIsSaved(), until a Save() call succeeds.
func (s *ServerState) SetCache(key string, obj any) {
	s.lock.Lock()
	defer s.lock.Unlock()
	encoded, err := json.Marshal(obj)
	if err != nil {
		panic("set cache error: " + err.Error())
	}
	s.cache[key] = encoded
	s.cacheIsSaved = false
}

// ClearCache clears the cache and toggles CacheIsSaved if necessary.
func (s *ServerState) ClearCache() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.cache) > 0 {
		s.cache = map[string]json.RawMessage{}
		s.cacheIsSaved = false
	}
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
		Cache       map[string]json.RawMessage
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
