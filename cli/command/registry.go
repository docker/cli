package command

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/credentials"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/hints"
	"github.com/docker/cli/cli/streams"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/registry"
)

const patSuggest = "You can log in with your password or a Personal Access " +
	"Token (PAT). Using a limited-scope PAT grants better security and is required " +
	"for organizations using SSO. Learn more at https://docs.docker.com/go/access-tokens/"

// RegistryAuthenticationPrivilegedFunc returns a RequestPrivilegeFunc from the specified registry index info
// for the given command.
func RegistryAuthenticationPrivilegedFunc(cli Cli, index *registrytypes.IndexInfo, cmdName string) registrytypes.RequestAuthConfig {
	return func(ctx context.Context) (string, error) {
		fmt.Fprintf(cli.Out(), "\nLogin prior to %s:\n", cmdName)
		indexServer := registry.GetAuthConfigKey(index)
		isDefaultRegistry := indexServer == registry.IndexServer
		authConfig, err := GetDefaultAuthConfig(cli.ConfigFile(), true, indexServer, isDefaultRegistry)
		if err != nil {
			fmt.Fprintf(cli.Err(), "Unable to retrieve stored credentials for %s, error: %s.\n", indexServer, err)
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
		configKey = registry.IndexServer
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
// Deprecated: use PromptUserForCredentials instead.
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
func PromptUserForCredentials(ctx context.Context, cli Cli, argUser, argPassword, defaultUsername, serverAddress string) (authConfig registrytypes.AuthConfig, err error) {
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

	isDefaultRegistry := serverAddress == registry.IndexServer
	defaultUsername = strings.TrimSpace(defaultUsername)

	if argUser = strings.TrimSpace(argUser); argUser == "" {
		if isDefaultRegistry {
			// if this is a default registry (docker hub), then display the following message.
			fmt.Fprintln(cli.Out(), "Log in with your Docker ID or email address to push and pull images from Docker Hub. If you don't have a Docker ID, head over to https://hub.docker.com/ to create one.")
			if hints.Enabled() {
				fmt.Fprintln(cli.Out(), patSuggest)
				fmt.Fprintln(cli.Out())
			}
		}

		var prompt string
		if defaultUsername == "" {
			prompt = "Username: "
		} else {
			prompt = fmt.Sprintf("Username (%s): ", defaultUsername)
		}
		argUser, err = PromptForInput(ctx, cli.In(), cli.Out(), prompt)
		if err != nil {
			return authConfig, err
		}
		if argUser == "" {
			argUser = defaultUsername
		}
	}
	if argUser == "" {
		return authConfig, fmt.Errorf("Error: Non-null Username Required")
	}
	if argPassword == "" {
		restoreInput, err := DisableInputEcho(cli.In())
		if err != nil {
			return authConfig, err
		}
		defer restoreInput()

		argPassword, err = PromptForInput(ctx, cli.In(), cli.Out(), "Password: ")
		if err != nil {
			return authConfig, err
		}
		fmt.Fprint(cli.Out(), "\n")
		if argPassword == "" {
			return authConfig, fmt.Errorf("Error: Password Required")
		}
	}

	authConfig.Username = argUser
	authConfig.Password = argPassword
	authConfig.ServerAddress = serverAddress
	return authConfig, nil
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

// resolveAuthConfigFromImage retrieves that AuthConfig using the image string
func resolveAuthConfigFromImage(cfg *configfile.ConfigFile, image string) (registrytypes.AuthConfig, error) {
	registryRef, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return registrytypes.AuthConfig{}, err
	}
	repoInfo, err := registry.ParseRepositoryInfo(registryRef)
	if err != nil {
		return registrytypes.AuthConfig{}, err
	}
	return ResolveAuthConfig(cfg, repoInfo.Index), nil
}
