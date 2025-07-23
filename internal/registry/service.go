package registry

import (
	"context"
	"crypto/tls"
	"errors"
	"net/url"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/docker/docker/api/types/registry"
)

// Service is a registry service. It tracks configuration data such as a list
// of mirrors.
type Service struct {
	config *serviceConfig
}

// NewService returns a new instance of [Service] ready to be installed into
// an engine.
func NewService(options ServiceOptions) (*Service, error) {
	config, err := newServiceConfig(options)
	if err != nil {
		return nil, err
	}

	return &Service{config: config}, err
}

// Auth contacts the public registry with the provided credentials,
// and returns OK if authentication was successful.
// It can be used to verify the validity of a client's credentials.
func (s *Service) Auth(ctx context.Context, authConfig *registry.AuthConfig, userAgent string) (statusMessage, token string, _ error) {
	// TODO Use ctx when searching for repositories
	registryHostName := IndexHostname

	if authConfig.ServerAddress != "" {
		serverAddress := authConfig.ServerAddress
		if !strings.HasPrefix(serverAddress, "https://") && !strings.HasPrefix(serverAddress, "http://") {
			serverAddress = "https://" + serverAddress
		}
		u, err := url.Parse(serverAddress)
		if err != nil {
			return "", "", invalidParamWrapf(err, "unable to parse server address")
		}
		registryHostName = u.Host
	}

	// Lookup endpoints for authentication but exclude mirrors to prevent
	// sending credentials of the upstream registry to a mirror.
	endpoints, err := s.lookupV2Endpoints(ctx, registryHostName, false)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", "", err
		}
		return "", "", invalidParam(err)
	}

	var lastErr error
	for _, endpoint := range endpoints {
		authToken, err := loginV2(ctx, authConfig, endpoint, userAgent)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || cerrdefs.IsUnauthorized(err) {
				// Failed to authenticate; don't continue with (non-TLS) endpoints.
				return "", "", err
			}
			// Try next endpoint
			log.G(ctx).WithFields(log.Fields{
				"error":    err,
				"endpoint": endpoint,
			}).Infof("Error logging in to endpoint, trying next endpoint")
			lastErr = err
			continue
		}

		// TODO(thaJeztah): move the statusMessage to the API endpoint; we don't need to produce that here?
		return "Login Succeeded", authToken, nil
	}

	return "", "", lastErr
}

// APIEndpoint represents a remote API endpoint
type APIEndpoint struct {
	Mirror    bool
	URL       *url.URL
	TLSConfig *tls.Config
}

// LookupPullEndpoints creates a list of v2 endpoints to try to pull from, in order of preference.
// It gives preference to mirrors over the actual registry, and HTTPS over plain HTTP.
func (s *Service) LookupPullEndpoints(hostname string) ([]APIEndpoint, error) {
	return s.lookupV2Endpoints(context.TODO(), hostname, true)
}

// LookupPushEndpoints creates a list of v2 endpoints to try to push to, in order of preference.
// It gives preference to HTTPS over plain HTTP. Mirrors are not included.
func (s *Service) LookupPushEndpoints(hostname string) ([]APIEndpoint, error) {
	return s.lookupV2Endpoints(context.TODO(), hostname, false)
}
