package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/unixpickle/lutronbroker/lutronbroker"
)

const (
	MinReauthInterval = time.Minute * 5
	ConnectionTimeout = time.Second * 10
)

type Server struct {
	state    *ServerState
	savePath string
	username string
	password string

	sessionLock   sync.RWMutex
	connection    BrokerConn
	reconnErr     error
	reconnErrTime *time.Time
}

func NewServer(savePath string, username string, password string) (*Server, error) {
	state, err := NewServerState(savePath)
	if err != nil {
		return nil, err
	}
	return &Server{
		state:    state,
		savePath: savePath,
		username: username,
		password: password,
	}, nil
}

func (s *Server) Serve(host string) error {
	s.addRoutes()
	log.Printf("listening on %s", host)
	return http.ListenAndServe(host, nil)
}

func (s *Server) addRoutes() {
	http.HandleFunc("/devices", s.serveDevices)
	http.HandleFunc("/clear_cache", s.serveClearCache)
}

func (s *Server) serveDevices(w http.ResponseWriter, r *http.Request) {
	s.handleGetCall(w, func(conn BrokerConn) (any, int, error) {
		devices, err := GetDevices(r.Context(), conn, s.state)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		return devices, http.StatusOK, nil
	})
}

func (s *Server) serveClearCache(w http.ResponseWriter, r *http.Request) {
	s.state.ClearCache()
	if !s.state.CacheIsSaved() {
		if err := s.state.Save(s.savePath); err != nil {
			serveError(w, http.StatusInternalServerError, err)
			return
		}
	}
	w.Header().Set("content-type", "application/json")
	w.Write([]byte(`{"data": true}`))
}

func (s *Server) handleGetCall(w http.ResponseWriter, f func(conn BrokerConn) (any, int, error)) {
	conn, err := s.getConnection()
	w.Header().Set("content-type", "application/json")
	if err != nil {
		serveError(w, http.StatusInternalServerError, err)
		return
	}
	obj, status, err := f(conn)
	if err != nil {
		serveError(w, status, err)
		return
	}
	data, err := json.Marshal(obj)
	if err != nil {
		serveError(w, http.StatusInternalServerError, err)
		return
	}
	if !s.state.CacheIsSaved() {
		if err := s.state.Save(s.savePath); err != nil {
			serveError(w, http.StatusInternalServerError, err)
			return
		}
	}
	w.WriteHeader(status)
	w.Write(data)
}

func (s *Server) getConnection() (conn BrokerConn, err error) {
	s.sessionLock.RLock()
	if s.connection != nil && s.connection.Error() == nil {
		s.sessionLock.RUnlock()
		return s.connection, nil
	}
	s.sessionLock.RUnlock()

	s.sessionLock.Lock()
	if s.reconnErr != nil && time.Since(*s.reconnErrTime) < MinReauthInterval {
		s.sessionLock.Unlock()
		return nil, s.reconnErr
	}
	defer func() {
		if err != nil {
			s.reconnErr = err
			t := time.Now()
			s.reconnErrTime = &t
		} else {
			s.connection = conn
		}
		s.sessionLock.Unlock()
	}()
	s.reconnErr = nil
	s.reconnErrTime = nil

	// Attempt to reauthenticate and reconnect.
	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	recreateCreds := func() (*lutronbroker.BrokerCredentials, error) {
		token, err := lutronbroker.GetOAuthToken(ctx, s.username, s.password)
		if err != nil {
			return nil, err
		}
		devices, err := lutronbroker.ListDevices(ctx, token)
		if err != nil {
			return nil, err
		}
		if len(devices) != 1 {
			return nil, fmt.Errorf("expected one device but found %d", len(devices))
		}
		device := devices[0]
		brokers, err := lutronbroker.ListDeviceBrokers(ctx, token, device.SerialNumber)
		if err != nil {
			return nil, err
		}
		if len(brokers) != 1 {
			return nil, fmt.Errorf("expected to find exactly one broker but found %d", len(brokers))
		}
		if len(brokers[0].AvailableBrokers) == 0 {
			return nil, errors.New("no available brokers found")
		}
		broker := brokers[0].AvailableBrokers[0]
		c, err := lutronbroker.AuthenticateWithBroker(ctx, token, device.SerialNumber, &broker)
		if err != nil {
			return nil, err
		}
		s.state.SetBrokerCreds(c)
		if err := s.state.Save(s.savePath); err != nil {
			return nil, err
		}
		return c, nil
	}

	creds := s.state.BrokerCreds()
	didAuth := creds == nil
	if didAuth {
		creds, err = recreateCreds()
		if err != nil {
			return nil, err
		}
	}

	conn, err = lutronbroker.NewBrokerConnection[Message](ctx, creds)
	if err == nil || didAuth {
		return
	}

	// Our session might have expired, so let's try reauthenticating.
	creds, err = recreateCreds()
	if err != nil {
		return nil, err
	}
	conn, err = lutronbroker.NewBrokerConnection[Message](ctx, creds)
	return
}

func serveError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	obj := map[string]string{"error": err.Error()}
	data, _ := json.Marshal(obj)
	w.Write(data)
}
