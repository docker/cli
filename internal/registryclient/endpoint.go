package registryclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/distribution/reference"
	"github.com/docker/cli/internal/registry"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/transport"
	registrytypes "github.com/moby/moby/api/types/registry"
)

type repositoryEndpoint struct {
	repoName  string
	indexInfo *registrytypes.IndexInfo
	endpoint  registry.APIEndpoint
	actions   []string
}

// BaseURL returns the endpoint url
func (r repositoryEndpoint) BaseURL() string {
	return r.endpoint.URL.String()
}

func newDefaultRepositoryEndpoint(ref reference.Named, insecure bool) (repositoryEndpoint, error) {
	indexInfo := registry.NewIndexInfo(ref)
	endpoint, err := getDefaultEndpoint(ref, !indexInfo.Secure)
	if err != nil {
		return repositoryEndpoint{}, err
	}
	if insecure {
		endpoint.TLSConfig.InsecureSkipVerify = true
	}
	return repositoryEndpoint{
		repoName:  reference.Path(reference.TrimNamed(ref)),
		indexInfo: indexInfo,
		endpoint:  endpoint,
	}, nil
}

func getDefaultEndpoint(repoName reference.Named, insecure bool) (registry.APIEndpoint, error) {
	registryService, err := registry.NewService(registry.ServiceOptions{})
	if err != nil {
		return registry.APIEndpoint{}, err
	}
	endpoints, err := registryService.Endpoints(context.TODO(), reference.Domain(repoName))
	if err != nil {
		return registry.APIEndpoint{}, err
	}
	// Default to the highest priority endpoint to return
	endpoint := endpoints[0]
	if insecure {
		for _, ep := range endpoints {
			if ep.URL.Scheme == "http" {
				endpoint = ep
			}
		}
	}
	return endpoint, nil
}

// getHTTPTransport builds a transport for use in communicating with a registry
func getHTTPTransport(authConfig registrytypes.AuthConfig, endpoint registry.APIEndpoint, repoName, userAgent string, actions []string) (http.RoundTripper, error) {
	// get the http transport, this will be used in a client to upload manifest
	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     endpoint.TLSConfig,
		DisableKeepAlives:   true,
	}

	modifiers := registry.Headers(userAgent, http.Header{})
	authTransport := transport.NewTransport(base, modifiers...)
	challengeManager, err := registry.PingV2Registry(endpoint.URL, authTransport)
	if err != nil {
		return nil, fmt.Errorf("error pinging v2 registry: %w", err)
	}
	if authConfig.RegistryToken != "" {
		passThruTokenHandler := &existingTokenHandler{token: authConfig.RegistryToken}
		modifiers = append(modifiers, auth.NewAuthorizer(challengeManager, passThruTokenHandler))
	} else {
		if len(actions) == 0 {
			actions = []string{"pull"}
		}
		creds := &staticCredentialStore{authConfig: &authConfig}
		tokenHandler := auth.NewTokenHandler(authTransport, creds, repoName, actions...)
		basicHandler := auth.NewBasicHandler(creds)
		modifiers = append(modifiers, auth.NewAuthorizer(challengeManager, tokenHandler, basicHandler))
	}
	return transport.NewTransport(base, modifiers...), nil
}

type existingTokenHandler struct {
	token string
}

func (th *existingTokenHandler) AuthorizeRequest(req *http.Request, _ map[string]string) error {
	req.Header.Set("Authorization", "Bearer "+th.token)
	return nil
}

func (*existingTokenHandler) Scheme() string {
	return "bearer"
}

type staticCredentialStore struct {
	authConfig *registrytypes.AuthConfig
}

func (scs staticCredentialStore) Basic(*url.URL) (string, string) {
	if scs.authConfig == nil {
		return "", ""
	}
	return scs.authConfig.Username, scs.authConfig.Password
}

func (scs staticCredentialStore) RefreshToken(*url.URL, string) string {
	if scs.authConfig == nil {
		return ""
	}
	return scs.authConfig.IdentityToken
}

func (staticCredentialStore) SetRefreshToken(*url.URL, string, string) {}
