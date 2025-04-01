package main

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/unixpickle/essentials"
	"github.com/unixpickle/lutronbroker/lutronbroker"
)

// ServerState is the state (e.g. session info) that the server saves across
// runs.
//
// Methods are safe to call concurrently from multiple Goroutines.
type ServerState struct {
	lock        sync.Mutex
	brokerCreds *lutronbroker.BrokerCredentials
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
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return &ServerState{brokerCreds: obj.BrokerCreds}, nil
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

// Save writes the state to a file.
func (s *ServerState) Save(path string) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	defer essentials.AddCtxTo("load server state", &err)

	var obj struct {
		BrokerCreds *lutronbroker.BrokerCredentials
	}
	obj.BrokerCreds = s.brokerCreds

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
