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
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("secret not found")
	ErrAccessDenied = errors.New("access denied") // nuh, uh, uh!
	ErrEmptySecret  = errors.New("secret value must not be empty")
)

type Envelope struct {
	ID         ID                `json:"-"`
	Value      []byte            `json:"-"`
	Metadata   map[string]string `json:"-"`
	Provider   string            `json:"-"`
	Version    string            `json:"-"`
	CreatedAt  time.Time         `json:"-"`
	ResolvedAt time.Time         `json:"-"`
	ExpiresAt  time.Time         `json:"-"`
}

var _ json.Marshaler = Envelope{}

func (e Envelope) MarshalJSON() ([]byte, error) {
	panic("secrets.Envelope does not support json.Marshal")
}

type Resolver interface {
	GetSecrets(ctx context.Context, pattern Pattern) ([]Envelope, error)
}

type AuthorizeResponse struct {
	// Expiry is when the decision stops holding. A zero Expiry means the
	// decision never expires, and the wire message then carries no timestamp.
	Expiry time.Time
	Allow  bool
}

type Authorizer interface {
	Authorize(ctx context.Context, pattern ...Pattern) (AuthorizeResponse, error)
}
