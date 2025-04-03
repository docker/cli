package server

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/docker/cli/cli/config/types"
)

const CredentialServerSocket = "docker_cli_credential_server.sock"

// GetCredentialServerSocket returns the path to the Unix socket
// configDir is the directory where the docker configuration file is stored
func GetCredentialServerSocket(configDir string) string {
	return filepath.Join(configDir, "run", CredentialServerSocket)
}

type CredentialConfig interface {
	GetAuthConfig(serverAddress string) (types.AuthConfig, error)
	GetAllCredentials() (map[string]types.AuthConfig, error)
}

// CheckCredentialServer checks if the credential server is running
// in the configDir directory by attempting to connect to the Unix socket.
// It returns the absolute path of the Unix socket if the server is running.
func CheckCredentialServer(configDir string) (string, error) {
	addr, err := net.ResolveUnixAddr("unix", GetCredentialServerSocket(configDir))
	if err != nil {
		return "", err
	}
	_, err = net.Dial(addr.Network(), addr.String())
	return addr.String(), err
}

// StartCredentialsServer hosts a Unix socket server that exposes
// the credentials store to the Docker CLI running in a container.
func StartCredentialsServer(ctx context.Context, configDir string, config CredentialConfig) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	l, err := net.ListenUnix("unix", &net.UnixAddr{
		Name: GetCredentialServerSocket(configDir),
		Net:  "unix",
	})
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/credentials", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			log.Println("GET /credentials")
			if key := r.URL.Query().Get("key"); key != "" {
				log.Printf("GET /credentials?key=%s", key)
				credential, err := config.GetAuthConfig(key)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := json.NewEncoder(w).Encode(credential); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				return
			}
			// Get credentials
			credentials, err := config.GetAllCredentials()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Write credentials
			err = json.NewEncoder(w).Encode(credentials)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case http.MethodPost:
			// Store credentials
		case http.MethodDelete:
			// Erase credentials
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	})

	timer := time.NewTimer(1000 * time.Second)
	activeConnections := atomic.Int32{}
	s := http.Server{
		BaseContext:  func(l net.Listener) context.Context { return ctx },
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
		ConnState: func(c net.Conn, cs http.ConnState) {
			switch cs {
			case http.StateActive, http.StateNew, http.StateHijacked:
				if activeConnections.Load() == 0 {
					timer.Stop()
				}
				activeConnections.Add(1)
			case http.StateClosed, http.StateIdle:
				if activeConnections.Load() == 0 {
					timer.Reset(10 * time.Second)
				}
				activeConnections.Add(-1)
			}
		},
		Handler: mux,
	}

	go func() {
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
		s.Shutdown(ctx)
	}()

	return s.Serve(l)
}
