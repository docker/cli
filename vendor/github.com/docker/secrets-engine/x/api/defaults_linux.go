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

//go:build linux

package api

import (
	"fmt"
	"os"
)

// StandaloneSocketPath returns the address of the standalone Secrets Engine's
// listening socket, @docker-secrets-engine/<UID>/daemon.sock.
// For example, UID 1000 uses @docker-secrets-engine/1000/daemon.sock.
//
// On Linux it is an abstract Unix domain socket: the address has a leading
// "@", which Go's net package maps to a NUL byte, placing the socket in the
// abstract namespace instead of on the filesystem.
//
// The address is namespaced by the user's UID so daemons run by different
// users on the same host do not collide (the abstract namespace is shared per
// network namespace, not per user as a filesystem path would be).
func StandaloneSocketPath() string {
	return fmt.Sprintf("@docker-secrets-engine/%d/daemon.sock", os.Getuid())
}

// DaemonSocketPath returns the address of the standalone Secrets Engine's
// listening socket.
//
// Deprecated: Use [StandaloneSocketPath] instead.
func DaemonSocketPath() string {
	return StandaloneSocketPath()
}
