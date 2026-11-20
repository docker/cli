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

package config

import (
	"os"
	"path/filepath"
)

// DesktopSocketPath returns the Secrets Engine socket path used by Docker Desktop.
//
//   - Linux: $XDG_CACHE_HOME/docker-secrets-engine/engine.sock, or
//     $HOME/.cache/docker-secrets-engine/engine.sock when XDG_CACHE_HOME is unset.
//   - macOS: $HOME/Library/Caches/docker-secrets-engine/engine.sock.
//   - Windows: %LOCALAPPDATA%\docker-secrets-engine\engine.sock.
//
// If the user cache directory cannot be determined, it returns
// <temp>/docker-secrets-engine/engine.sock, where <temp> is the directory returned
// by [os.TempDir].
func DesktopSocketPath() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "docker-secrets-engine", "engine.sock")
	}
	return filepath.Join(os.TempDir(), "docker-secrets-engine", "engine.sock")
}
