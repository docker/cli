package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/commands"
	"github.com/docker/cli/internal/oauth/manager"
	"github.com/docker/cli/internal/registry"
	"github.com/docker/cli/internal/tui"
	registrytypes "github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	commands.Register(newLoginCommand)
}

type loginOptions struct {
	serverAddress string
	user          string
	password      string
	passwordStdin bool
}

// newLoginCommand creates a new `docker login` command
func newLoginCommand(dockerCLI command.Cli) *cobra.Command {
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
			if err := verifyLoginFlags(cmd.Flags(), opts); err != nil {
				return err
			}
			return runLogin(cmd.Context(), dockerCLI, opts)
		},
		Annotations: map[string]string{
			"category-top": "8",
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.user, "username", "u", "", "Username")
	flags.StringVarP(&opts.password, "password", "p", "", "Password or Personal Access Token (PAT)")
	flags.BoolVar(&opts.passwordStdin, "password-stdin", false, "Take the Password or Personal Access Token (PAT) from stdin")

	return cmd
}

// verifyLoginFlags validates flags set on the command.
//
// TODO(thaJeztah); combine with verifyLoginOptions, but this requires rewrites of many tests.
func verifyLoginFlags(flags *pflag.FlagSet, opts loginOptions) error {
	if flags.Changed("password-stdin") {
		if flags.Changed("password") {
			return errors.New("conflicting options: cannot specify both --password and --password-stdin")
		}
		if !flags.Changed("username") {
			return errors.New("the --password-stdin option requires --username to be set")
		}
	}
	if flags.Changed("username") && opts.user == "" {
		return errors.New("username is empty")
	}
	if flags.Changed("password") && opts.password == "" {
		return errors.New("password is empty")
	}
	return nil
}

func verifyLoginOptions(dockerCLI command.Streams, opts *loginOptions) error {
	if opts.password != "" {
		_, _ = fmt.Fprintln(dockerCLI.Err(), "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
	}

	if opts.passwordStdin {
		if opts.user == "" {
			return errors.New("username is empty")
		}

		contents, err := io.ReadAll(dockerCLI.In())
		if err != nil {
			return err
		}

		opts.password = strings.TrimSuffix(string(contents), "\n")
		opts.password = strings.TrimSuffix(opts.password, "\r")
	}
	return nil
}

func runLogin(ctx context.Context, dockerCLI command.Cli, opts loginOptions) error {
	if err := verifyLoginOptions(dockerCLI, &opts); err != nil {
		return err
	}

	maybePrintEnvAuthWarning(dockerCLI)

	var serverAddress string
	if opts.serverAddress != "" && opts.serverAddress != registry.DefaultNamespace {
		serverAddress = opts.serverAddress
	} else {
		serverAddress = registry.IndexServer
	}
	isDefaultRegistry := serverAddress == registry.IndexServer

	// attempt login with current (stored) credentials
	authConfig, err := command.GetDefaultAuthConfig(dockerCLI.ConfigFile(), opts.user == "" && opts.password == "", serverAddress, isDefaultRegistry)
	if err == nil && authConfig.Username != "" && authConfig.Password != "" {
		err = loginWithStoredCredentials(ctx, dockerCLI, authConfig)
	}

	// if we failed to authenticate with stored credentials (or didn't have stored credentials),
	// prompt the user for new credentials
	if err != nil || authConfig.Username == "" || authConfig.Password == "" {
		err = loginUser(ctx, dockerCLI, opts, authConfig.Username, authConfig.ServerAddress)
		if err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), "Login Succeeded")
	return nil
}

func loginWithStoredCredentials(ctx context.Context, dockerCLI command.Cli, authConfig registrytypes.AuthConfig) error {
	_, _ = fmt.Fprintf(dockerCLI.Err(), "Authenticating with existing credentials...")
	if authConfig.Username != "" {
		_, _ = fmt.Fprintf(dockerCLI.Err(), " [Username: %s]", authConfig.Username)
	}
	_, _ = fmt.Fprint(dockerCLI.Err(), "\n")

	out := tui.NewOutput(dockerCLI.Err())
	out.PrintNote("To login with a different account, run 'docker logout' followed by 'docker login'")

	_, _ = fmt.Fprint(dockerCLI.Err(), "\n\n")

	response, err := dockerCLI.Client().RegistryLogin(ctx, authConfig)
	if err != nil {
		if errdefs.IsUnauthorized(err) {
			_, _ = fmt.Fprintln(dockerCLI.Err(), "Stored credentials invalid or expired")
		} else {
			_, _ = fmt.Fprintln(dockerCLI.Err(), "Login did not succeed, error:", err)
		}
		return err
	}

	if response.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = response.IdentityToken
	}

	return storeCredentials(dockerCLI.ConfigFile(), authConfig)
}

// OauthLoginEscapeHatchEnvVar disables the browser-based OAuth login workflow.
//
// Deprecated: this const was only used internally and will be removed in the next release.
const OauthLoginEscapeHatchEnvVar = "DOCKER_CLI_DISABLE_OAUTH_LOGIN"

const oauthLoginEscapeHatchEnvVar = "DOCKER_CLI_DISABLE_OAUTH_LOGIN"

func isOauthLoginDisabled() bool {
	if v := os.Getenv(oauthLoginEscapeHatchEnvVar); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return enabled
	}
	return false
}

func loginUser(ctx context.Context, dockerCLI command.Cli, opts loginOptions, defaultUsername, serverAddress string) error {
	// Some links documenting this:
	// - https://code.google.com/archive/p/mintty/issues/56
	// - https://github.com/docker/docker/issues/15272
	// - https://mintty.github.io/ (compatibility)
	// Linux will hit this if you attempt `cat | docker login`, and Windows
	// will hit this if you attempt docker login from mintty where stdin
	// is a pipe, not a character based console.
	if (opts.user == "" || opts.password == "") && !dockerCLI.In().IsTerminal() {
		return errors.New("error: cannot perform an interactive login from a non-TTY device")
	}

	// If we're logging into the index server and the user didn't provide a username or password, use the device flow
	if serverAddress == registry.IndexServer && opts.user == "" && opts.password == "" && !isOauthLoginDisabled() {
		err := loginWithDeviceCodeFlow(ctx, dockerCLI)
		// if the error represents a failure to initiate the device-code flow,
		// then we fallback to regular cli credentials login
		if !errors.Is(err, manager.ErrDeviceLoginStartFail) {
			return err
		}
		_, _ = fmt.Fprint(dockerCLI.Err(), "Failed to start web-based login - falling back to command line login...\n\n")
	}

	return loginWithUsernameAndPassword(ctx, dockerCLI, opts, defaultUsername, serverAddress)
}

func loginWithUsernameAndPassword(ctx context.Context, dockerCLI command.Cli, opts loginOptions, defaultUsername, serverAddress string) error {
	// Prompt user for credentials
	authConfig, err := command.PromptUserForCredentials(ctx, dockerCLI, opts.user, opts.password, defaultUsername, serverAddress)
	if err != nil {
		return err
	}

	response, err := loginWithRegistry(ctx, dockerCLI.Client(), authConfig)
	if err != nil {
		return err
	}

	if response.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = response.IdentityToken
	}
	return storeCredentials(dockerCLI.ConfigFile(), authConfig)
}

func loginWithDeviceCodeFlow(ctx context.Context, dockerCLI command.Cli) error {
	store := dockerCLI.ConfigFile().GetCredentialsStore(registry.IndexServer)
	authConfig, err := manager.NewManager(store).LoginDevice(ctx, dockerCLI.Err())
	if err != nil {
		return err
	}

	_, err = loginWithRegistry(ctx, dockerCLI.Client(), registrytypes.AuthConfig(*authConfig))
	if err != nil {
		return err
	}

	return storeCredentials(dockerCLI.ConfigFile(), registrytypes.AuthConfig(*authConfig))
}

func storeCredentials(cfg *configfile.ConfigFile, authConfig registrytypes.AuthConfig) error {
	creds := cfg.GetCredentialsStore(authConfig.ServerAddress)
	if err := creds.Store(configtypes.AuthConfig(authConfig)); err != nil {
		return fmt.Errorf("error saving credentials: %v", err)
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

	token, err := svc.Auth(ctx, &auth, command.UserAgent())
	if err != nil {
		return nil, err
	}

	return &registrytypes.AuthenticateOKBody{
		IdentityToken: token,
	}, nil
}
