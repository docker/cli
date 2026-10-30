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

// Package realms keeps track of known Docker realms
//
// Realms do not define a permission model and should not be used as such!
// Realms are simply contracts that clients may use to query a set of secrets.
// Once a realm has been established it MUST not change as clients would treat
// the realm as a contract.
// Clients may pin themselves to a specific realm or a group of realms.
//
// Examples:
//
//	`docker/` is a realm for all known Docker secrets.
//	`docker/auth` is a realm for all known Docker Auth secrets.
package realms
