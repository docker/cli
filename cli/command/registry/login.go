package registry

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/internal/oauth/manager"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type loginOptions struct {
	serverAddress string
	user          string
	password      string
	passwordStdin bool
}

// NewLoginCommand creates a new `docker login` command
func NewLoginCommand(dockerCli command.Cli) *cobra.Command {
	var opts loginOptions

	cmd := &cobra.Command{
		Use:   "login [OPTIONS] [SERVER]",
		Short: "Authenticate to a registry",
		Long:  "Authenticate to a registry.\nDefaults to Docker Hub if no server is specified.",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.serverAddress = args[0]
			}
			return runLogin(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"category-top": "8",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.user, "username", "u", "", "Username")
	flags.StringVarP(&opts.password, "password", "p", "", "Password")
	flags.BoolVar(&opts.passwordStdin, "password-stdin", false, "Take the password from stdin")

	return cmd
}

func verifyLoginOptions(dockerCli command.Cli, opts *loginOptions) error {
	if opts.password != "" {
		_, _ = fmt.Fprintln(dockerCli.Err(), "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
		if opts.passwordStdin {
			return errors.New("--password and --password-stdin are mutually exclusive")
		}
	}

	if opts.passwordStdin {
		if opts.user == "" {
			return errors.New("Must provide --username with --password-stdin")
		}

		contents, err := io.ReadAll(dockerCli.In())
		if err != nil {
			return err
		}

		opts.password = strings.TrimSuffix(string(contents), "\n")
		opts.password = strings.TrimSuffix(opts.password, "\r")
	}
	return nil
}

func runLogin(ctx context.Context, dockerCli command.Cli, opts loginOptions) error {
	if err := verifyLoginOptions(dockerCli, &opts); err != nil {
		return err
	}
	var (
		serverAddress string
		msg           string
	)
	if opts.serverAddress != "" && opts.serverAddress != registry.DefaultNamespace {
		serverAddress = opts.serverAddress
	} else {
		serverAddress = registry.IndexServer
	}
	isDefaultRegistry := serverAddress == registry.IndexServer

	// attempt login with current (stored) credentials
	authConfig, err := command.GetDefaultAuthConfig(dockerCli.ConfigFile(), opts.user == "" && opts.password == "", serverAddress, isDefaultRegistry)
	if err == nil && authConfig.Username != "" && authConfig.Password != "" {
		msg, err = loginWithStoredCredentials(ctx, dockerCli, authConfig)
	}

	// if we failed to authenticate with stored credentials (or didn't have stored credentials),
	// prompt the user for new credentials
	if err != nil || authConfig.Username == "" || authConfig.Password == "" {
		msg, err = loginUser(ctx, dockerCli, opts, authConfig.Username, authConfig.ServerAddress)
		if err != nil {
			return err
		}
	}

	if msg != "" {
		_, _ = fmt.Fprintln(dockerCli.Out(), msg)
	}
	return nil
}

func loginWithStoredCredentials(ctx context.Context, dockerCli command.Cli, authConfig registrytypes.AuthConfig) (msg string, _ error) {
	_, _ = fmt.Fprintf(dockerCli.Out(), "Authenticating with existing credentials...\n")
	response, err := dockerCli.Client().RegistryLogin(ctx, authConfig)
	if err != nil {
		if errdefs.IsUnauthorized(err) {
			_, _ = fmt.Fprintf(dockerCli.Err(), "Stored credentials invalid or expired\n")
		} else {
			_, _ = fmt.Fprintf(dockerCli.Err(), "Login did not succeed, error: %s\n", err)
		}
	}

	if response.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = response.IdentityToken
	}

	if err := storeCredentials(dockerCli.ConfigFile(), authConfig); err != nil {
		return "", err
	}

	return response.Status, err
}

const OauthLoginEscapeHatchEnvVar = "DOCKER_CLI_DISABLE_OAUTH_LOGIN"

func isOauthLoginDisabled() bool {
	if v := os.Getenv(OauthLoginEscapeHatchEnvVar); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return enabled
	}
	return false
}

func loginUser(ctx context.Context, dockerCli command.Cli, opts loginOptions, defaultUsername, serverAddress string) (msg string, _ error) {
	// Some links documenting this:
	// - https://code.google.com/archive/p/mintty/issues/56
	// - https://github.com/docker/docker/issues/15272
	// - https://mintty.github.io/ (compatibility)
	// Linux will hit this if you attempt `cat | docker login`, and Windows
	// will hit this if you attempt docker login from mintty where stdin
	// is a pipe, not a character based console.
	if (opts.user == "" || opts.password == "") && !dockerCli.In().IsTerminal() {
		return "", errors.Errorf("Error: Cannot perform an interactive login from a non TTY device")
	}

	// If we're logging into the index server and the user didn't provide a username or password, use the device flow
	if serverAddress == registry.IndexServer && opts.user == "" && opts.password == "" && !isOauthLoginDisabled() {
		var err error
		msg, err = loginWithDeviceCodeFlow(ctx, dockerCli)
		// if the error represents a failure to initiate the device-code flow,
		// then we fallback to regular cli credentials login
		if !errors.Is(err, manager.ErrDeviceLoginStartFail) {
			return msg, err
		}
		_, _ = fmt.Fprint(dockerCli.Err(), "Failed to start web-based login - falling back to command line login...\n\n")
	}

	return loginWithUsernameAndPassword(ctx, dockerCli, opts, defaultUsername, serverAddress)
}

func loginWithUsernameAndPassword(ctx context.Context, dockerCli command.Cli, opts loginOptions, defaultUsername, serverAddress string) (msg string, _ error) {
	// Prompt user for credentials
	authConfig, err := command.PromptUserForCredentials(ctx, dockerCli, opts.user, opts.password, defaultUsername, serverAddress)
	if err != nil {
		return "", err
	}

	response, err := loginWithRegistry(ctx, dockerCli.Client(), authConfig)
	if err != nil {
		return "", err
	}

	if response.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = response.IdentityToken
	}
	if err = storeCredentials(dockerCli.ConfigFile(), authConfig); err != nil {
		return "", err
	}

	return response.Status, nil
}

func loginWithDeviceCodeFlow(ctx context.Context, dockerCli command.Cli) (msg string, _ error) {
	store := dockerCli.ConfigFile().GetCredentialsStore(registry.IndexServer)
	authConfig, err := manager.NewManager(store).LoginDevice(ctx, dockerCli.Err())
	if err != nil {
		return "", err
	}

	response, err := loginWithRegistry(ctx, dockerCli.Client(), registrytypes.AuthConfig(*authConfig))
	if err != nil {
		return "", err
	}

	if err = storeCredentials(dockerCli.ConfigFile(), registrytypes.AuthConfig(*authConfig)); err != nil {
		return "", err
	}

	return response.Status, nil
}

func storeCredentials(cfg *configfile.ConfigFile, authConfig registrytypes.AuthConfig) error {
	creds := cfg.GetCredentialsStore(authConfig.ServerAddress)
	if err := creds.Store(configtypes.AuthConfig(authConfig)); err != nil {
		return errors.Errorf("Error saving credentials: %v", err)
	}

	return nil
}

func loginWithRegistry(ctx context.Context, apiClient client.SystemAPIClient, authConfig registrytypes.AuthConfig) (*registrytypes.AuthenticateOKBody, error) {
	response, err := apiClient.RegistryLogin(ctx, authConfig)
	if err != nil {
		if client.IsErrConnectionFailed(err) {
			// daemon isn't responding; attempt to login client side.
			return loginClientSide(ctx, authConfig)
		}
		return nil, err
	}

	return &response, nil
}

func loginClientSide(ctx context.Context, auth registrytypes.AuthConfig) (*registrytypes.AuthenticateOKBody, error) {
	svc, err := registry.NewService(registry.ServiceOptions{})
	if err != nil {
		return nil, err
	}

	status, token, err := svc.Auth(ctx, &auth, command.UserAgent())
	if err != nil {
		return nil, err
	}

	return &registrytypes.AuthenticateOKBody{
		Status:        status,
		IdentityToken: token,
	}, nil
}
