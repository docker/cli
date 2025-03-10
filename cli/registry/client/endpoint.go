package client

import (
	"net"
	"net/http"
	"time"

	"github.com/distribution/reference"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/transport"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
)

type repositoryEndpoint struct {
	info     *registry.RepositoryInfo
	endpoint registry.APIEndpoint
	actions  []string
}

// Name returns the repository name
func (r repositoryEndpoint) Name() string {
	return reference.Path(r.info.Name)
}

// BaseURL returns the endpoint url
func (r repositoryEndpoint) BaseURL() string {
	return r.endpoint.URL.String()
}

func newDefaultRepositoryEndpoint(ref reference.Named, insecure bool) (repositoryEndpoint, error) {
	repoInfo, _ := registry.ParseRepositoryInfo(ref)
	endpoint, err := getDefaultEndpointFromRepoInfo(repoInfo)
	if err != nil {
		return repositoryEndpoint{}, err
	}
	if insecure {
		endpoint.TLSConfig.InsecureSkipVerify = true
	}
	return repositoryEndpoint{info: repoInfo, endpoint: endpoint}, nil
}

func getDefaultEndpointFromRepoInfo(repoInfo *registry.RepositoryInfo) (registry.APIEndpoint, error) {
	var err error

	options := registry.ServiceOptions{}
	registryService, err := registry.NewService(options)
	if err != nil {
		return registry.APIEndpoint{}, err
	}
	endpoints, err := registryService.LookupPushEndpoints(reference.Domain(repoInfo.Name))
	if err != nil {
		return registry.APIEndpoint{}, err
	}
	// Default to the highest priority endpoint to return
	endpoint := endpoints[0]
	if !repoInfo.Index.Secure {
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
		return nil, errors.Wrap(err, "error pinging v2 registry")
	}
	if authConfig.RegistryToken != "" {
		passThruTokenHandler := &existingTokenHandler{token: authConfig.RegistryToken}
		modifiers = append(modifiers, auth.NewAuthorizer(challengeManager, passThruTokenHandler))
	} else {
		if len(actions) == 0 {
			actions = []string{"pull"}
		}
		creds := registry.NewStaticCredentialStore(&authConfig)
		tokenHandler := auth.NewTokenHandler(authTransport, creds, repoName, actions...)
		basicHandler := auth.NewBasicHandler(creds)
		modifiers = append(modifiers, auth.NewAuthorizer(challengeManager, tokenHandler, basicHandler))
	}
	return transport.NewTransport(base, modifiers...), nil
}

// RepoNameForReference returns the repository name from a reference.
//
// Deprecated: this function is no longer used and will be removed in the next release.
func RepoNameForReference(ref reference.Named) (string, error) {
	return reference.Path(reference.TrimNamed(ref)), nil
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
