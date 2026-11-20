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

// Package secrets defines the core types shared across the Secrets Engine:
// secret identifiers and patterns, resolved secret envelopes, and the
// interfaces implemented by secret stores.
//
// # Identifiers and patterns
//
// An [ID] contains a secret identifier. Valid secret identifiers must match
// the format ^[A-Za-z0-9._:-]+(?:/[A-Za-z0-9._:-]+)*$:
//
//   - Components separated by '/'
//   - Each component is non-empty
//   - Only characters A-Z, a-z, 0-9, '.', '_', '-' or ':'
//   - No leading, trailing, or double slashes
//
// For storage, we don't really differentiate much about the ID format but
// by convention we do simple, slash-separated management, providing a
// groupable access control system for management across plugins.
//
// A [Pattern] matches secret IDs. It follows the ID validation rules, except
// that '*' matches one component and '**' matches zero or more.
//
//	id := secrets.MustParseID("docker/auth/hub/joe")
//	pattern := secrets.MustParsePattern("docker/auth/**")
//	pattern.Match(id) // true
//
// # Resolving secrets
//
// A [github.com/docker/secrets-engine/client/api/resolver.Resolver] returns
// the [Envelope] of every secret that matches a pattern. A
// [github.com/docker/secrets-engine/client/api/resolver.Authorizer] decides
// whether access to a set of patterns is allowed.
//
// # Storing secrets
//
// A [Store] persists values implementing [Secret], keyed by [ID]. Store
// backends take a [Factory] that creates an empty [Secret] for each ID they
// read, so callers decide how their secret values are encoded.
package secrets
