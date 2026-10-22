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

// Package realms exposes the canonical set of Docker realm patterns that
// clients use to scope secret queries.
//
// A realm is a path-pattern contract - not a permission boundary. Clients use
// realms to declare which set of secrets they care about (e.g. Hub auth,
// registry auth, MCP OAuth). Once a realm is established its pattern MUST NOT
// change; clients may pin to a realm and treat it as a stable interface.
//
// The Docker realm hierarchy is:
//
//	docker/                        – all Docker secrets
//	docker/auth/hub/**             – Docker Hub authentication (OAuth login)
//	docker/auth/hub-staging/**     – Docker Hub staging authentication
//	docker/auth/registry/docker/** – Docker Registry authentication
//	docker/auth/metadata/hub/**    – Hub account profile metadata
//	docker/mcp/**                  – MCP-related secrets
//	docker/mcp/oauth/**            – MCP OAuth credentials
//	docker/mcp/oauth-dcr/**        – MCP Dynamic Client Registration configs
//	docker/sandbox/**              – Sandbox-related secrets
//	docker/sandbox/oauth/**        – Sandbox third-party OAuth tokens
//
// All variables in this package are re-exported from
// github.com/docker/secrets-engine/x/realms and are provided here as a
// stable, versioned surface for external consumers.
package realms

import xrealms "github.com/docker/secrets-engine/x/realms"

// Docker realms all start with `docker/` as the prefix.
//
// Authentication flows done by the Docker CLI, Docker Desktop and Docker related
// products must go through `docker/auth`.
//
// Docker Hub authentication (browser based OAuth login) will be prefixed
// with `docker/auth/hub/<username>`.
//
// Docker Registry authentication will be prefixed with
// `docker/auth/registry/docker/<username>`.
var (
	DockerHubAuthentication             = xrealms.DockerHubAuthentication
	DockerHubStagingAuthentication      = xrealms.DockerHubStagingAuthentication
	DockerRegistryAuthentication        = xrealms.DockerRegistryAuthentication
	DockerRegistryStagingAuthentication = xrealms.DockerRegistryStagingAuthentication
)

var (
	// DockerHubAuthenticationMetadata holds one profile entry per signed-in
	// Docker Hub account, plus a default entry duplicating the default
	// account's profile.
	DockerHubAuthenticationMetadata = xrealms.DockerHubAuthenticationMetadata
	// DockerHubStagingAuthenticationMetadata is the staging variant of
	// [DockerHubAuthenticationMetadata].
	DockerHubStagingAuthenticationMetadata = xrealms.DockerHubStagingAuthenticationMetadata
)

var (
	// DockerMCPDefault is the default realm used for MCP related secrets
	DockerMCPDefault = xrealms.DockerMCPDefault
	// DockerMCPOAuth is the realm used for all MCP OAuth credentials retrieved by Docker.
	DockerMCPOAuth = xrealms.DockerMCPOAuth
	// DockerMCPOAuthDCR is the realm used to hold Dynamic Client Registered (DCR)
	// OAuth configurations for supported MCP servers.
	DockerMCPOAuthDCR = xrealms.DockerMCPOAuthDCR
)

var (
	// DockerSandbox is the default realm used for Sandbox related secrets
	DockerSandbox = xrealms.DockerSandbox
	// DockerSandboxOAuth is the realm used for all Sandbox OAuth credentials
	// such as third-party tokens - this does not store the Docker Auth tokens.
	DockerSandboxOAuth = xrealms.DockerSandboxOAuth
)
