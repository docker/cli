// Copyright 2026 Docker, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"path/filepath"

	"github.com/docker/secrets-engine/x/api"
)

var (
	ErrSecretsEngineNotRunning       = errors.New("secrets engine is not running")
	ErrSecretsEnginePermissionDenied = errors.New("permission denied connecting to the secrets engine")
	ErrSecretsEngineTimeout          = errors.New("secrets engine did not respond in time")
)

type ConnectReason int

const (
	ReasonUnknown ConnectReason = iota
	ReasonNotRunning
	ReasonPermissionDenied
	ReasonTimeout
)

func (r ConnectReason) String() string {
	switch r {
	case ReasonNotRunning:
		return "not running"
	case ReasonPermissionDenied:
		return "permission denied"
	case ReasonTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

func (r ConnectReason) sentinel() error {
	switch r {
	case ReasonNotRunning:
		return ErrSecretsEngineNotRunning
	case ReasonPermissionDenied:
		return ErrSecretsEnginePermissionDenied
	case ReasonTimeout:
		return ErrSecretsEngineTimeout
	default:
		return nil
	}
}

type ConnectError struct {
	Reason     ConnectReason
	SocketPath string
	Err        error
}

func (e *ConnectError) Error() string {
	msg := "cannot connect to the secrets engine"
	if s := e.Reason.sentinel(); s != nil {
		msg = s.Error()
	}
	if e.SocketPath != "" {
		msg += " at " + e.SocketPath
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

func (e *ConnectError) Unwrap() error { return e.Err }

func (e *ConnectError) Is(target error) bool {
	s := e.Reason.sentinel()
	return s != nil && target == s
}

const (
	desktopNotRunningHint    = "Start Docker Desktop and retry: the secrets engine runs as part of it."
	desktopTimeoutHint       = "The secrets engine did not respond. Restart Docker Desktop and retry."
	standaloneNotRunningHint = "Start the standalone secrets engine and retry: nothing is listening on %s."
	standaloneTimeoutHint    = "The standalone secrets engine on %s did not respond. Restart it and retry."
	customNotRunningHint     = "Start the secrets engine and retry: nothing is listening on %s."
	customTimeoutHint        = "The secrets engine on %s did not respond. Restart it and retry."
	permissionDeniedHint     = "Your user lacks permission to connect to %s. Grant it read and write access to the socket and retry."
)

// Hint returns a user friendly message to help recover from the error.
func (e *ConnectError) Hint() string {
	socket := "the secrets engine socket"
	if e.SocketPath != "" {
		socket = fmt.Sprintf("%q", e.SocketPath)
	}
	kind := classifySocket(e.SocketPath)
	switch e.Reason {
	case ReasonNotRunning:
		switch kind {
		case engineDesktop:
			return desktopNotRunningHint
		case engineStandalone:
			return fmt.Sprintf(standaloneNotRunningHint, socket)
		default:
			return fmt.Sprintf(customNotRunningHint, socket)
		}
	case ReasonTimeout:
		switch kind {
		case engineDesktop:
			return desktopTimeoutHint
		case engineStandalone:
			return fmt.Sprintf(standaloneTimeoutHint, socket)
		default:
			return fmt.Sprintf(customTimeoutHint, socket)
		}
	case ReasonPermissionDenied:
		return fmt.Sprintf(permissionDeniedHint, socket)
	default:
		return ""
	}
}

type engineKind int

const (
	engineCustom engineKind = iota
	engineDesktop
	engineStandalone
)

func classifySocket(path string) engineKind {
	if path == "" {
		return engineCustom
	}
	switch filepath.Clean(path) {
	case filepath.Clean(api.DesktopSocketPath()):
		return engineDesktop
	case filepath.Clean(api.StandaloneSocketPath()):
		return engineStandalone
	default:
		return engineCustom
	}
}

func connectReason(err error) ConnectReason {
	var ne net.Error
	switch {
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, errConnRefused):
		return ReasonNotRunning
	case errors.Is(err, fs.ErrPermission):
		return ReasonPermissionDenied
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &ne) && ne.Timeout():
		return ReasonTimeout
	default:
		return ReasonUnknown
	}
}
