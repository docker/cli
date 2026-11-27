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

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when no secret exists for the requested ID or
	// pattern.
	ErrNotFound = errors.New("secret not found")
	// ErrAccessDenied is returned when the caller is not allowed to access
	// the requested secrets.
	ErrAccessDenied = errors.New("access denied") // nuh, uh, uh!
	// ErrEmptySecret is returned when a secret with an empty value is saved.
	ErrEmptySecret = errors.New("secret value must not be empty")
)

// Envelope is a resolved secret together with information about where it
// came from and how long it is valid.
//
// An Envelope carries plaintext secret material and cannot be encoded with
// encoding/json; see [Envelope.MarshalJSON].
type Envelope struct {
	// ID identifies the secret.
	ID ID `json:"-"`
	// Value is the plaintext secret value.
	Value []byte `json:"-"`
	// Metadata holds non-sensitive key-value data about the secret.
	Metadata map[string]string `json:"-"`
	// Provider names the provider that resolved the secret.
	Provider string `json:"-"`
	// Version is the version of the payload format in Value, set by the
	// provider. It is independent of the provider's own release version.
	Version string `json:"-"`
	// CreatedAt is when the secret was created.
	CreatedAt time.Time `json:"-"`
	// ResolvedAt is when the secret was resolved.
	ResolvedAt time.Time `json:"-"`
	// ExpiresAt is when the secret stops being valid.
	ExpiresAt time.Time `json:"-"`
}

var _ json.Marshaler = Envelope{}

// MarshalJSON always panics, so that an Envelope and its plaintext value are
// never written out by encoding/json by accident.
func (e Envelope) MarshalJSON() ([]byte, error) {
	panic("secrets.Envelope does not support json.Marshal")
}
