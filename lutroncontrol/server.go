package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/unixpickle/lutronbroker/lutronbroker"
)

const (
	MinReauthInterval = time.Minute * 5
	ConnectionTimeout = time.Second * 10
	PingInterval      = time.Second * 20
	PingTimeout       = time.Second * 5
)

type Server struct {
	state    *ServerState
	assetDir string
	savePath string
	username string
	password string
	basePath string

	sessionLock   sync.RWMutex
	connection    BrokerConn
	reconnErr     error
	reconnErrTime *time.Time
}

func NewServer(assetDir, savePath string, username string, password string, basePath string) (*Server, error) {
	state, err := NewServerState(savePath)
	if err != nil {
		return nil, err
	}
	if basePath == "" {
		basePath = "/"
	}
	if basePath[0] != '/' {
		basePath = "/" + basePath
	}
	if len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}
	return &Server{
		state:    state,
		assetDir: assetDir,
		savePath: savePath,
		username: username,
		password: password,
		basePath: basePath,
	}, nil
}

func (s *Server) Serve(host string) error {
	mux := s.addRoutes()
	log.Printf("listening on %s", host)
	return http.ListenAndServe(host, mux)
}

func (s *Server) addRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(s.assetDir))
	if s.basePath == "/" {
		mux.Handle("/", fs)
		mux.HandleFunc("/devices", s.serveDevices)
		mux.HandleFunc("/clear_cache", s.serveClearCache)
		mux.HandleFunc("/command/all_off", s.serveAllOff)
		mux.HandleFunc("/command/set_level", s.serveSetLevel)
		mux.HandleFunc("/command/press_and_release", s.servePressAndRelease)
		mux.HandleFunc("/scenes", s.serveScenes)
		mux.HandleFunc("/scene/activate", s.serveSceneActivate)
		mux.HandleFunc("/scene/activate_by_name", s.serveSceneActivateByName)
	} else {
		mux.Handle(s.basePath+"/", http.StripPrefix(s.basePath+"/", fs))
		mux.HandleFunc(s.basePath+"/devices", s.serveDevices)
		mux.HandleFunc(s.basePath+"/clear_cache", s.serveClearCache)
		mux.HandleFunc(s.basePath+"/command/all_off", s.serveAllOff)
		mux.HandleFunc(s.basePath+"/command/set_level", s.serveSetLevel)
		mux.HandleFunc(s.basePath+"/command/press_and_release", s.servePressAndRelease)
		mux.HandleFunc(s.basePath+"/scenes", s.serveScenes)
		mux.HandleFunc(s.basePath+"/scene/activate", s.serveSceneActivate)
		mux.HandleFunc(s.basePath+"/scene/activate_by_name", s.serveSceneActivateByName)
	}
	return mux
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

func (s *Server) serveAllOff(w http.ResponseWriter, r *http.Request) {
	s.handleGetCall(w, func(conn BrokerConn) (any, int, error) {
		devices, err := GetDevices(r.Context(), conn, s.state)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		for _, device := range devices {
			if device.Zone == nil || *device.Zone == "" {
				continue
			}
			if device.DeviceType == "QsWirelessShade" {
				continue
			}
			commandType := "GoToDimmedLevel"
			command := map[string]any{"CommandType": commandType}
			if device.DeviceType == "WallSwitch" {
				commandType = "GoToSwitchedLevel"
				command["CommandType"] = commandType
				command["SwitchedLevelParameters"] = map[string]any{
					"SwitchedLevel": "Off",
				}
			} else {
				command["DimmedLevelParameters"] = map[string]any{
					"Level": 0,
				}
			}
			body := map[string]any{"Command": command}
			if err := CreateRequest(r.Context(), conn, *device.Zone+"/commandprocessor", body); err != nil {
				return nil, http.StatusInternalServerError, err
			}
		}
		return map[string]bool{"data": true}, http.StatusOK, nil
	})
}

func (s *Server) serveSetLevel(w http.ResponseWriter, r *http.Request) {
	s.handleGetCall(w, func(conn BrokerConn) (any, int, error) {
		commandType := r.FormValue("type")
		zone := r.FormValue("zone")
		if _, err := strconv.Atoi(zone); err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid zone: %w", err)
		}

		command := map[string]any{"CommandType": commandType}

		if commandType == "Raise" || commandType == "Lower" || commandType == "Stop" {
			// No additional parameters needed for these shade commands.
		} else {
			levelStr := r.FormValue("level")
			level, err := strconv.Atoi(levelStr)
			if err == nil && (level < 0 || level > 100) {
				err = errors.New("level is out of range")
			}
			if err != nil {
				return nil, http.StatusBadRequest, fmt.Errorf("invalid level: %w", err)
			}

			if commandType == "GoToLevel" {
				command["Parameter"] = map[string]any{
					"Type":  "Level",
					"Value": level,
				}
			} else if commandType == "GoToDimmedLevel" {
				command["DimmedLevelParameters"] = map[string]any{
					"Level": level,
				}
			} else if commandType == "GoToSwitchedLevel" {
				name := "On"
				if level == 0 {
					name = "Off"
				}
				command["SwitchedLevelParameters"] = map[string]any{
					"SwitchedLevel": name,
				}
			} else {
				return nil, http.StatusBadRequest, fmt.Errorf("unknown command type: %s", commandType)
			}
		}

		body := map[string]any{"Command": command}
		if err := CreateRequest(r.Context(), conn, "/zone/"+zone+"/commandprocessor", body); err == nil {
			return true, http.StatusOK, nil
		} else {
			return false, http.StatusInternalServerError, err
		}
	})
}

func (s *Server) servePressAndRelease(w http.ResponseWriter, r *http.Request) {
	s.handleGetCall(w, func(conn BrokerConn) (any, int, error) {
		button := r.FormValue("button")
		if _, err := strconv.Atoi(button); err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid button: %w", err)
		}

		body := map[string]any{
			"Command": map[string]any{
				"CommandType": "PressAndRelease",
			},
		}
		if err := CreateRequest(r.Context(), conn, "/button/"+button+"/commandprocessor", body); err == nil {
			return true, http.StatusOK, nil
		} else {
			return false, http.StatusInternalServerError, err
		}
	})
}

func (s *Server) serveScenes(w http.ResponseWriter, r *http.Request) {
	s.handleGetCall(w, func(conn BrokerConn) (any, int, error) {
		var response struct {
			VirtualButtons []struct {
				Href         string `json:"href"`
				Name         string
				IsProgrammed bool
				ButtonNumber int
			}
		}
		if err := ReadRequest(r.Context(), conn, "/virtualbutton", &response); err != nil {
			return nil, http.StatusInternalServerError, err
		}
		return response.VirtualButtons, http.StatusOK, nil
	})
}

func (s *Server) serveSceneActivate(w http.ResponseWriter, r *http.Request) {
	s.handleGetCall(w, func(conn BrokerConn) (any, int, error) {
		scene := r.FormValue("scene")
		if _, err := strconv.Atoi(scene); err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid scene: %w", err)
		}
		body := map[string]any{
			"Command": map[string]any{
				"CommandType": "PressAndRelease",
			},
		}
		if err := CreateRequest(r.Context(), conn, "/virtualbutton/"+scene+"/commandprocessor", body); err == nil {
			return true, http.StatusOK, nil
		} else {
			return false, http.StatusInternalServerError, err
		}
	})
}

func (s *Server) serveSceneActivateByName(w http.ResponseWriter, r *http.Request) {
	s.handleGetCall(w, func(conn BrokerConn) (any, int, error) {
		sceneName := r.FormValue("name")
		if sceneName == "" {
			return map[string]bool{"data": false}, http.StatusOK, nil
		}
		var response struct {
			VirtualButtons []struct {
				Href         string `json:"href"`
				Name         string
				IsProgrammed bool
			}
		}
		if err := ReadRequest(r.Context(), conn, "/virtualbutton", &response); err != nil {
			return nil, http.StatusInternalServerError, err
		}
		var href string
		for _, scene := range response.VirtualButtons {
			if !scene.IsProgrammed {
				continue
			}
			if strings.EqualFold(scene.Name, sceneName) {
				href = scene.Href
				break
			}
		}
		if href == "" {
			return map[string]bool{"data": false}, http.StatusOK, nil
		}
		body := map[string]any{
			"Command": map[string]any{
				"CommandType": "PressAndRelease",
			},
		}
		if err := CreateRequest(r.Context(), conn, href+"/commandprocessor", body); err == nil {
			return map[string]bool{"data": true}, http.StatusOK, nil
		} else {
			return nil, http.StatusInternalServerError, err
		}
	})
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

	// Check if some other goroutine in getConnection raced us.
	if s.connection != nil && s.connection.Error() == nil {
		s.sessionLock.Unlock()
		return s.connection, nil
	}

	if err := s.connection.Error(); err != nil {
		log.Println("reconnecting due to connection error:", err)
	}

	if s.reconnErr != nil && time.Since(*s.reconnErrTime) < MinReauthInterval {
		s.sessionLock.Unlock()
		return nil, s.reconnErr
	}
	defer func() {
		if err != nil {
			log.Println("error establishing new broker connection:", err)
			s.reconnErr = err
			t := time.Now()
			s.reconnErrTime = &t
		} else {
			log.Println("established new broker connection")
			s.connection = conn
			go s.pingLoop(conn)
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

func (s *Server) pingLoop(conn BrokerConn) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()
	for range ticker.C {
		if conn.Error() != nil {
			// No need to clear the connection; future getConnection()
			// calls will replace it.
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), PingTimeout)
		var response any
		err := ReadRequest(ctx, conn, "/server/1/status/ping", &response)
		cancel()
		if conn.Error() != nil {
			// See comment above; this is handled already.
			return
		} else if err != nil {
			// There's some error (e.g. a timeout) that the BrokerConn missed.
			s.sessionLock.Lock()
			defer s.sessionLock.Unlock()
			// Make sure the connection didn't hit an error right before we got
			// the lock, in which case it could have already been replaced.
			if s.connection == conn {
				log.Println("disconnecting due to ping failure:", err)
				s.connection = nil
				conn.Close()
			}
			return
		}
	}
}

func serveError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	obj := map[string]string{"error": err.Error()}
	data, _ := json.Marshal(obj)
	w.Write(data)
}
