package command

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"golang.org/x/net/context"

	"github.com/docker/cli/registry"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/term"
	dockerregistry "github.com/docker/docker/registry"
	"github.com/pkg/errors"
)

// RegistryAuthenticationPrivilegedFunc returns a RequestPrivilegeFunc from the specified registry index info
// for the given command.
func RegistryAuthenticationPrivilegedFunc(cli Cli, index *registrytypes.IndexInfo, cmdName string) types.RequestPrivilegeFunc {
	return func() (string, error) {
		fmt.Fprintf(cli.Out(), "\nPlease login prior to %s:\n", cmdName)
		indexServer := dockerregistry.GetAuthConfigKey(index)
		electedServer, warns, err := registry.ElectAuthServer(context.Background(), cli.Client())
		for _, w := range warns {
			fmt.Fprintf(cli.Err(), "Warning: %v\n", w)
		}
		if err != nil {
			return "", err
		}
		isDefaultRegistry := indexServer == electedServer
		authConfig, err := ConfigureAuth(cli, "", "", indexServer, isDefaultRegistry)
		if err != nil {
			return "", err
		}
		return registry.EncodeAuthToBase64(authConfig)
	}
}

// ConfigureAuth returns an AuthConfig from the specified user, password and server.
func ConfigureAuth(cli Cli, flUser, flPassword, serverAddress string, isDefaultRegistry bool) (types.AuthConfig, error) {
	// On Windows, force the use of the regular OS stdin stream. Fixes #14336/#14210
	if runtime.GOOS == "windows" {
		cli.SetIn(NewInStream(os.Stdin))
	}

	if !isDefaultRegistry {
		serverAddress = dockerregistry.ConvertToHostname(serverAddress)
	}

	authconfig, err := cli.CredentialsStore(serverAddress).Get(serverAddress)
	if err != nil {
		return authconfig, err
	}

	// Some links documenting this:
	// - https://code.google.com/archive/p/mintty/issues/56
	// - https://github.com/docker/docker/issues/15272
	// - https://mintty.github.io/ (compatibility)
	// Linux will hit this if you attempt `cat | docker login`, and Windows
	// will hit this if you attempt docker login from mintty where stdin
	// is a pipe, not a character based console.
	if flPassword == "" && !cli.In().IsTerminal() {
		return authconfig, errors.Errorf("Error: Cannot perform an interactive login from a non TTY device")
	}

	authconfig.Username = strings.TrimSpace(authconfig.Username)

	if flUser = strings.TrimSpace(flUser); flUser == "" {
		if isDefaultRegistry {
			// if this is a default registry (docker hub), then display the following message.
			fmt.Fprintln(cli.Out(), "Login with your Docker ID to push and pull images from Docker Hub. If you don't have a Docker ID, head over to https://hub.docker.com to create one.")
		}
		promptWithDefault(cli.Out(), "Username", authconfig.Username)
		flUser = readInput(cli.In(), cli.Out())
		flUser = strings.TrimSpace(flUser)
		if flUser == "" {
			flUser = authconfig.Username
		}
	}
	if flUser == "" {
		return authconfig, errors.Errorf("Error: Non-null Username Required")
	}
	if flPassword == "" {
		oldState, err := term.SaveState(cli.In().FD())
		if err != nil {
			return authconfig, err
		}
		fmt.Fprintf(cli.Out(), "Password: ")
		term.DisableEcho(cli.In().FD(), oldState)

		flPassword = readInput(cli.In(), cli.Out())
		fmt.Fprint(cli.Out(), "\n")

		term.RestoreTerminal(cli.In().FD(), oldState)
		if flPassword == "" {
			return authconfig, errors.Errorf("Error: Password Required")
		}
	}

	authconfig.Username = flUser
	authconfig.Password = flPassword
	authconfig.ServerAddress = serverAddress
	authconfig.IdentityToken = ""

	return authconfig, nil
}

func readInput(in io.Reader, out io.Writer) string {
	reader := bufio.NewReader(in)
	line, _, err := reader.ReadLine()
	if err != nil {
		fmt.Fprintln(out, err.Error())
		os.Exit(1)
	}
	return string(line)
}

func promptWithDefault(out io.Writer, prompt string, configDefault string) {
	if configDefault == "" {
		fmt.Fprintf(out, "%s: ", prompt)
	} else {
		fmt.Fprintf(out, "%s (%s): ", prompt, configDefault)
	}
}

// RetrieveAuthTokenFromImage retrieves an encoded auth token given a complete image
func RetrieveAuthTokenFromImage(ctx context.Context, cli Cli, image string) (string, error) {
	// Retrieve encoded auth token from the image reference
	authConfig, err := resolveAuthConfigFromImage(ctx, cli, image)
	if err != nil {
		return "", err
	}
	encodedAuth, err := registry.EncodeAuthToBase64(authConfig)
	if err != nil {
		return "", err
	}
	return encodedAuth, nil
}

// resolveAuthConfigFromImage retrieves that AuthConfig using the image string
func resolveAuthConfigFromImage(ctx context.Context, cli Cli, image string) (types.AuthConfig, error) {
	registryRef, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return types.AuthConfig{}, err
	}
	repoInfo, err := dockerregistry.ParseRepositoryInfo(registryRef)
	if err != nil {
		return types.AuthConfig{}, err
	}
	ac, warns, err := registry.ResolveAuthConfig(ctx, cli.Client(), cli.ConfigFile(), repoInfo.Index)
	for _, w := range warns {
		fmt.Fprintf(cli.Err(), "Warning: %v\n", w)
	}
	return ac, err
}
