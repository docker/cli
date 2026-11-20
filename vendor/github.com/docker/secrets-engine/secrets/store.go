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

package secrets

import "context"

// Secret is a generic type that represents the actual secret values
//
// The implementer is responsible for defining the data structure of their secrets.
//
// Example:
//
//	type secret struct {
//		AccessToken string
//		RefreshToken string
//	}
//
//	func (s *secret) Marshal() ([]byte, error) {
//		return []byte(s.AccessToken+":"+s.RefreshToken), nil
//	}
//
//	func (s *secret) Unmarshal(data []byte) error {
//		tokens := bytes.Split(data, []byte(":"))
//		if len(tokens) != 2 {
//			return errors.New("invalid secret format")
//		}
//
//		s.AccessToken, s.RefreshToken = string(tokens[0]), string(tokens[1])
//		return nil
//	}
type Secret interface {
	// Marshal the secret into a slice of bytes
	Marshal() ([]byte, error)
	// Unmarshal the secret from a slice of bytes into its structured format.
	// Implementations must copy any bytes they retain: store backends zero
	// the source buffer after Unmarshal returns to avoid leaking plaintext.
	Unmarshal(data []byte) error
	// Metadata returns a key-value pair of non-sensitive data about the secret
	Metadata() map[string]string
	// SetMetadata allows the caller to set the secrets non-sensitive data
	// A secret may expects certain keys or values from the map and may return
	// an error.
	SetMetadata(map[string]string) error
}

// Store defines a strict format for secrets to conform to when interacting
// with the secrets engine
type Store interface {
	// Delete removes credentials from the store for a given ID.
	Delete(ctx context.Context, id ID) error
	// Get retrieves credentials from the store for a given ID.
	Get(ctx context.Context, id ID) (Secret, error)
	// GetAllMetadata retrieves all the credentials from the store.
	// Credentials retrieved will only call [Secret.SetMetadata] so that the
	// underlying store does not get queried for each secret's sensitive data.
	// This could be very taxing on the underlying store and cause a poor User
	// Experience.
	GetAllMetadata(ctx context.Context) (map[ID]Secret, error)
	// Save persists credentials from the store.
	//
	// An empty secret value is rejected with [ErrEmptySecret].
	Save(ctx context.Context, id ID, secret Secret) error
	// Upsert atomically replaces an existing credential or inserts a new one.
	//
	// An empty secret value is rejected with [ErrEmptySecret].
	//
	// On the macOS Keychain an existing item is updated in place, keeping the
	// applications the user allowed to access it. Another store instance or
	// process can still write the same item concurrently, in which case Upsert
	// returns a backend-specific duplicate item error.
	Upsert(ctx context.Context, id ID, secret Secret) error
	// Filter returns a map of secrets based on a [Pattern].
	//
	// A pattern matching nothing is a valid empty result: implementations
	// return an empty (non-nil) map with a nil error, not ErrNotFound.
	//
	// Secrets returned will have both [Secret.SetMetadata] and [Secret.Unmarshal]
	// called; in that order. Any error produced by any of them would result in
	// an early return with a nil secrets map.
	Filter(ctx context.Context, pattern Pattern) (map[ID]Secret, error)
}

// Factory returns a new, empty [Secret] for the given ID. Store backends call
// it for every secret they read, then fill it in with [Secret.SetMetadata]
// and, when the value is read too, [Secret.Unmarshal].
type Factory[T Secret] func(context.Context, ID) T
