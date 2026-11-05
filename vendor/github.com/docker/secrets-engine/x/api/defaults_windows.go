// Copyright 2025-2026 Docker, Inc.
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

//go:build windows

package api

import (
	"os"
	"path/filepath"
)

// StandaloneSocketPath returns the standalone Secrets Engine's listening socket
// path, %LOCALAPPDATA%\DockerSecretsEngine\service\daemon.sock.
// If LOCALAPPDATA is unset, it uses
// %USERPROFILE%\AppData\Local\DockerSecretsEngine\service\daemon.sock.
func StandaloneSocketPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return filepath.Join(base, "DockerSecretsEngine", "service", "daemon.sock")
}

// DaemonSocketPath returns the standalone Secrets Engine's listening socket path.
//
// Deprecated: Use [StandaloneSocketPath] instead.
func DaemonSocketPath() string {
	return StandaloneSocketPath()
}
