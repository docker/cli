package command

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/credentials"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/hints"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/tui"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/morikuni/aec"
	"github.com/pkg/errors"
)

const (
	registerSuggest = "Log in with your Docker ID or email address to push and pull images from Docker Hub. " +
		"If you don't have a Docker ID, head over to https://hub.docker.com/ to create one."
	patSuggest = "You can log in with your password or a Personal Access " +
		"Token (PAT). Using a limited-scope PAT grants better security and is required " +
		"for organizations using SSO. Learn more at https://docs.docker.com/go/access-tokens/"
)

const (
	// IndexHostname is the index hostname, used for authentication and image search.
	indexHostname = "index.docker.io"
	// IndexServer is used for user auth and image search
	indexServer = "https://" + indexHostname + "/v1/"
)

// RegistryAuthenticationPrivilegedFunc returns a RequestPrivilegeFunc from the specified registry index info
// for the given command.
func RegistryAuthenticationPrivilegedFunc(cli Cli, index *registrytypes.IndexInfo, cmdName string) registrytypes.RequestAuthConfig {
	return func(ctx context.Context) (string, error) {
		_, _ = fmt.Fprintf(cli.Out(), "\nLogin prior to %s:\n", cmdName)
		isDefaultRegistry := index.Official || index.Name == indexServer
		authConfig, err := GetDefaultAuthConfig(cli.ConfigFile(), true, indexServer, isDefaultRegistry)
		if err != nil {
			_, _ = fmt.Fprintf(cli.Err(), "Unable to retrieve stored credentials for %s, error: %s.\n", indexServer, err)
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		authConfig, err = PromptUserForCredentials(ctx, cli, "", "", authConfig.Username, indexServer)
		if err != nil {
			return "", err
		}
		return registrytypes.EncodeAuthConfig(authConfig)
	}
}

// ResolveAuthConfig returns auth-config for the given registry from the
// credential-store. It returns an empty AuthConfig if no credentials were
// found.
//
// It is similar to [registry.ResolveAuthConfig], but uses the credentials-
// store, instead of looking up credentials from a map.
func ResolveAuthConfig(cfg *configfile.ConfigFile, index *registrytypes.IndexInfo) registrytypes.AuthConfig {
	configKey := index.Name
	if index.Official {
		configKey = indexServer
	}

	a, _ := cfg.GetAuthConfig(configKey)
	return registrytypes.AuthConfig(a)
}

// GetDefaultAuthConfig gets the default auth config given a serverAddress
// If credentials for given serverAddress exists in the credential store, the configuration will be populated with values in it
func GetDefaultAuthConfig(cfg *configfile.ConfigFile, checkCredStore bool, serverAddress string, isDefaultRegistry bool) (registrytypes.AuthConfig, error) {
	if !isDefaultRegistry {
		serverAddress = credentials.ConvertToHostname(serverAddress)
	}
	authconfig := configtypes.AuthConfig{}
	var err error
	if checkCredStore {
		authconfig, err = cfg.GetAuthConfig(serverAddress)
		if err != nil {
			return registrytypes.AuthConfig{
				ServerAddress: serverAddress,
			}, err
		}
	}
	authconfig.ServerAddress = serverAddress
	authconfig.IdentityToken = ""
	return registrytypes.AuthConfig(authconfig), nil
}

// ConfigureAuth handles prompting of user's username and password if needed.
//
// Deprecated: use [PromptUserForCredentials] instead.
func ConfigureAuth(ctx context.Context, cli Cli, flUser, flPassword string, authConfig *registrytypes.AuthConfig, _ bool) error {
	defaultUsername := authConfig.Username
	serverAddress := authConfig.ServerAddress

	newAuthConfig, err := PromptUserForCredentials(ctx, cli, flUser, flPassword, defaultUsername, serverAddress)
	if err != nil {
		return err
	}

	authConfig.Username = newAuthConfig.Username
	authConfig.Password = newAuthConfig.Password
	return nil
}

// PromptUserForCredentials handles the CLI prompt for the user to input
// credentials.
// If argUser is not empty, then the user is only prompted for their password.
// If argPassword is not empty, then the user is only prompted for their username
// If neither argUser nor argPassword are empty, then the user is not prompted and
// an AuthConfig is returned with those values.
// If defaultUsername is not empty, the username prompt includes that username
// and the user can hit enter without inputting a username  to use that default
// username.
func PromptUserForCredentials(ctx context.Context, cli Cli, argUser, argPassword, defaultUsername, serverAddress string) (registrytypes.AuthConfig, error) {
	// On Windows, force the use of the regular OS stdin stream.
	//
	// See:
	// - https://github.com/moby/moby/issues/14336
	// - https://github.com/moby/moby/issues/14210
	// - https://github.com/moby/moby/pull/17738
	//
	// TODO(thaJeztah): we need to confirm if this special handling is still needed, as we may not be doing this in other places.
	if runtime.GOOS == "windows" {
		cli.SetIn(streams.NewIn(os.Stdin))
	}

	argUser = strings.TrimSpace(argUser)
	if argUser == "" {
		if serverAddress == indexServer {
			// When signing in to the default (Docker Hub) registry, we display
			// hints for creating an account, and (if hints are enabled), using
			// a token instead of a password.
			_, _ = fmt.Fprintln(cli.Out(), registerSuggest)
			if hints.Enabled() {
				_, _ = fmt.Fprintln(cli.Out(), patSuggest)
				_, _ = fmt.Fprintln(cli.Out())
			}
		}

		var prompt string
		defaultUsername = strings.TrimSpace(defaultUsername)
		if defaultUsername == "" {
			prompt = "Username: "
		} else {
			prompt = fmt.Sprintf("Username (%s): ", defaultUsername)
		}

		var err error
		argUser, err = PromptForInput(ctx, cli.In(), cli.Out(), prompt)
		if err != nil {
			return registrytypes.AuthConfig{}, err
		}
		if argUser == "" {
			argUser = defaultUsername
		}
		if argUser == "" {
			return registrytypes.AuthConfig{}, errors.Errorf("Error: Non-null Username Required")
		}
	}

	argPassword = strings.TrimSpace(argPassword)
	if argPassword == "" {
		restoreInput, err := DisableInputEcho(cli.In())
		if err != nil {
			return registrytypes.AuthConfig{}, err
		}
		defer func() {
			if err := restoreInput(); err != nil {
				// TODO(thaJeztah): we should consider printing instructions how
				//  to restore this manually (other than restarting the shell).
				//  e.g., 'run stty echo' when in a Linux or macOS shell, but
				//  PowerShell and CMD.exe may need different instructions.
				_, _ = fmt.Fprintln(cli.Err(), "Error: failed to restore terminal state to echo input:", err)
			}
		}()

		out := tui.NewOutput(cli.Err())
		out.PrintNote("A Personal Access Token (PAT) can be used instead.\n" +
			"To create a PAT, visit " + aec.Underline.Apply("https://app.docker.com/settings") + "\n\n")
		argPassword, err = PromptForInput(ctx, cli.In(), cli.Out(), "Password: ")
		if err != nil {
			return registrytypes.AuthConfig{}, err
		}
		_, _ = fmt.Fprintln(cli.Out())
		if argPassword == "" {
			return registrytypes.AuthConfig{}, errors.Errorf("Error: Password Required")
		}
	}

	return registrytypes.AuthConfig{
		Username:      argUser,
		Password:      argPassword,
		ServerAddress: serverAddress,
	}, nil
}

// RetrieveAuthTokenFromImage retrieves an encoded auth token given a complete
// image. The auth configuration is serialized as a base64url encoded RFC4648,
// section 5) JSON string for sending through the X-Registry-Auth header.
//
// For details on base64url encoding, see:
// - RFC4648, section 5:   https://tools.ietf.org/html/rfc4648#section-5
func RetrieveAuthTokenFromImage(cfg *configfile.ConfigFile, image string) (string, error) {
	// Retrieve encoded auth token from the image reference
	authConfig, err := resolveAuthConfigFromImage(cfg, image)
	if err != nil {
		return "", err
	}
	encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
	if err != nil {
		return "", err
	}
	return encodedAuth, nil
}

func normalizeIndexName(val string) string {
	// TODO(thaJeztah): consider normalizing other known options, such as "(https://)registry-1.docker.io", "https://index.docker.io/v1/".
	// TODO: upstream this to check to reference package
	if val == "index.docker.io" {
		return "docker.io"
	}
	return val
}

// newIndexInfo returns IndexInfo configuration from indexName
func newIndexInfo(indexName string) *registrytypes.IndexInfo {
	indexName = normalizeIndexName(indexName)

	// Return any configured index info, first.
	if indexName == "docker.io" {
		return &registrytypes.IndexInfo{
			Name:     "docker.io",
			Secure:   true,
			Official: true,
		}
	}

	// Construct a non-configured index info.
	return &registrytypes.IndexInfo{
		Name:    indexName,
		Mirrors: []string{},
		Secure:  !isInsecure(indexName),
	}
}

// resolveAuthConfigFromImage retrieves that AuthConfig using the image string
func resolveAuthConfigFromImage(cfg *configfile.ConfigFile, image string) (registrytypes.AuthConfig, error) {
	registryRef, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return registrytypes.AuthConfig{}, err
	}
	domainName := reference.Domain(registryRef)
	idxInfo := newIndexInfo(domainName)
	return ResolveAuthConfig(cfg, idxInfo), nil
}

// isInsecure is used to detect whether a registry domain or IP-address is allowed
// to use an insecure (non-TLS, or self-signed cert) connection according to the
// defaults, which allows for insecure connections with registries running on a
// loopback address ("localhost", "::1/128", "127.0.0.0/8").
//
// It is used in situations where we don't have access to the daemon's configuration,
// for example, when used from the client / CLI.
func isInsecure(hostNameOrIP string) bool {
	// Attempt to strip port if present; this also strips brackets for
	// IPv6 addresses with a port (e.g. "[::1]:5000").
	//
	// This is best-effort; we'll continue using the address as-is if it fails.
	if host, _, err := net.SplitHostPort(hostNameOrIP); err == nil {
		hostNameOrIP = host
	}
	if hostNameOrIP == "127.0.0.1" || hostNameOrIP == "::1" || strings.EqualFold(hostNameOrIP, "localhost") {
		// Fast path; no need to resolve these, assuming nobody overrides
		// "localhost" for anything else than a loopback address (sorry, not sorry).
		return true
	}

	var addresses []net.IP
	if ip := net.ParseIP(hostNameOrIP); ip != nil {
		addresses = append(addresses, ip)
	} else {
		// Try to resolve the host's IP-addresses.
		addrs, _ := net.LookupIP(hostNameOrIP)
		addresses = append(addresses, addrs...)
	}

	for _, addr := range addresses {
		if addr.IsLoopback() {
			return true
		}
	}
	return false
}
