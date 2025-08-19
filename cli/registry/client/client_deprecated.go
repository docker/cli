package client // Deprecated: this package was only used internally and will be removed in the next release.

import "github.com/docker/cli/internal/registryclient"

// RegistryClient is a client used to communicate with a Docker distribution
// registry.
//
// Deprecated: this interface was only used internally and will be removed in the next release.
type RegistryClient = registryclient.RegistryClient

// NewRegistryClient returns a new RegistryClient with a resolver
//
// Deprecated: this function was only used internally and will be removed in the next release.
func NewRegistryClient(resolver registryclient.AuthConfigResolver, userAgent string, insecure bool) registryclient.RegistryClient {
	return registryclient.NewRegistryClient(resolver, userAgent, insecure)
}

// AuthConfigResolver returns Auth Configuration for an index
//
// Deprecated: this type was only used internally and will be removed in the next release.
type AuthConfigResolver = registryclient.AuthConfigResolver
