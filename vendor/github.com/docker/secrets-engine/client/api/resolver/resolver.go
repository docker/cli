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

package resolver

import (
	"context"
	"time"

	"github.com/docker/secrets-engine/secrets"
)

// Resolver resolves secrets by pattern.
type Resolver interface {
	// GetSecrets returns an Envelope for every secret matching pattern.
	GetSecrets(ctx context.Context, pattern secrets.Pattern) ([]secrets.Envelope, error)
}

// AuthorizeResponse is the access decision returned by an [Authorizer].
type AuthorizeResponse struct {
	// Expiry is when the decision stops holding. A zero Expiry means the
	// decision never expires, and the wire message then carries no timestamp.
	Expiry time.Time
	// Allow reports whether access is granted. A deny decision is returned
	// with Allow false and a nil error.
	Allow bool
}

// Authorizer decides access to secrets.
type Authorizer interface {
	// Authorize decides access to all of the given patterns at once.
	Authorize(ctx context.Context, pattern ...secrets.Pattern) (AuthorizeResponse, error)
}
