package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	authConfig, err := command.GetDefaultAuthConfig(dockerCLI.ConfigFile(), opts.user == "" && opts.password == "", serverAddress, isDefaultRegistry)
	if err == nil && authConfig.Username != "" && authConfig.Password != "" {
		msg, err = loginWithStoredCredentials(ctx, dockerCLI, authConfig)
	}

	// if we failed to authenticate with stored credentials (or didn't have stored credentials),
	// prompt the user for new credentials
	if err != nil || authConfig.Username == "" || authConfig.Password == "" {
		msg, err = loginUser(ctx, dockerCLI, opts, authConfig.Username, authConfig.ServerAddress)
		if err != nil {
			return err
		}
	}

	if msg != "" {
		_, _ = fmt.Fprintln(dockerCLI.Out(), msg)
	}
	return nil
}

func loginWithStoredCredentials(ctx context.Context, dockerCLI command.Cli, authConfig registrytypes.AuthConfig) (msg string, _ error) {
	_, _ = fmt.Fprintf(dockerCLI.Err(), "Authenticating with existing credentials...")
	if authConfig.Username != "" {
		_, _ = fmt.Fprintf(dockerCLI.Err(), " [Username: %s]", authConfig.Username)
	}
	_, _ = fmt.Fprint(dockerCLI.Err(), "\n")

	out := tui.NewOutput(dockerCLI.Err())
	out.PrintNote("To login with a different account, run 'docker logout' followed by 'docker login'")

	_, _ = fmt.Fprint(dockerCLI.Err(), "\n\n")

	resp, err := dockerCLI.Client().RegistryLogin(ctx, client.RegistryLoginOptions{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	})
	if err != nil {
		if errdefs.IsUnauthorized(err) {
			_, _ = fmt.Fprintln(dockerCLI.Err(), "Stored credentials invalid or expired")
		} else {
			_, _ = fmt.Fprintln(dockerCLI.Err(), "Login did not succeed, error:", err)
		}
		// TODO(thaJeztah): should this return the error here, or is there a reason for continuing?
	}

	if resp.Auth.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = resp.Auth.IdentityToken
	}

	if err := storeCredentials(dockerCLI.ConfigFile(), authConfig); err != nil {
		return "", err
	}

	return resp.Auth.Status, err
}

func loginUser(ctx context.Context, dockerCLI command.Cli, opts loginOptions, defaultUsername, serverAddress string) (msg string, _ error) {
	// Some links documenting this:
	// - https://code.google.com/archive/p/mintty/issues/56
	// - https://github.com/docker/docker/issues/15272
	// - https://mintty.github.io/ (compatibility)
	// Linux will hit this if you attempt `cat | docker login`, and Windows
	// will hit this if you attempt docker login from mintty where stdin
	// is a pipe, not a character based console.
	if (opts.user == "" || opts.password == "") && !dockerCLI.In().IsTerminal() {
		return "", errors.New("error: cannot perform an interactive login from a non TTY device")
	}

	// If we're logging into the index server and the user didn't provide a username or password, use the device flow
	if serverAddress == registry.IndexServer && opts.user == "" && opts.password == "" {
		var err error
		msg, err = loginWithDeviceCodeFlow(ctx, dockerCLI)
		// if the error represents a failure to initiate the device-code flow,
		// then we fallback to regular cli credentials login
		if !errors.Is(err, manager.ErrDeviceLoginStartFail) {
			return msg, err
		}
		_, _ = fmt.Fprint(dockerCLI.Err(), "Failed to start web-based login - falling back to command line login...\n\n")
	}

	return loginWithUsernameAndPassword(ctx, dockerCLI, opts, defaultUsername, serverAddress)
}

func loginWithUsernameAndPassword(ctx context.Context, dockerCLI command.Cli, opts loginOptions, defaultUsername, serverAddress string) (msg string, _ error) {
	// Prompt user for credentials
	authConfig, err := command.PromptUserForCredentials(ctx, dockerCLI, opts.user, opts.password, defaultUsername, serverAddress)
	if err != nil {
		return "", err
	}

	res, err := loginWithRegistry(ctx, dockerCLI.Client(), client.RegistryLoginOptions{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	})
	if err != nil {
		return "", err
	}

	if res.Auth.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = res.Auth.IdentityToken
	}
	if err = storeCredentials(dockerCLI.ConfigFile(), authConfig); err != nil {
		return "", err
	}

	return res.Auth.Status, nil
}

func loginWithDeviceCodeFlow(ctx context.Context, dockerCLI command.Cli) (msg string, _ error) {
	store := dockerCLI.ConfigFile().GetCredentialsStore(registry.IndexServer)
	authConfig, err := manager.NewManager(store).LoginDevice(ctx, dockerCLI.Err())
	if err != nil {
		return "", err
	}

	response, err := loginWithRegistry(ctx, dockerCLI.Client(), client.RegistryLoginOptions{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		ServerAddress: authConfig.ServerAddress,

		// TODO(thaJeztah): Are these expected to be included?
		// Auth:          authConfig.Auth,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	})
	if err != nil {
		return "", err
	}

	if err = storeCredentials(dockerCLI.ConfigFile(), registrytypes.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		ServerAddress: authConfig.ServerAddress,

		// TODO(thaJeztah): Are these expected to be included?
		Auth:          authConfig.Auth,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}); err != nil {
		return "", err
	}

	return response.Auth.Status, nil
}

func storeCredentials(cfg *configfile.ConfigFile, authConfig registrytypes.AuthConfig) error {
	creds := cfg.GetCredentialsStore(authConfig.ServerAddress)
	if err := creds.Store(configtypes.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		ServerAddress: authConfig.ServerAddress,

		// TODO(thaJeztah): Are these expected to be included?
		Auth:          authConfig.Auth,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}); err != nil {
		return fmt.Errorf("error saving credentials: %v", err)
	}

	return nil
}

func loginWithRegistry(ctx context.Context, apiClient client.SystemAPIClient, options client.RegistryLoginOptions) (client.RegistryLoginResult, error) {
	res, err := apiClient.RegistryLogin(ctx, options)
	if err != nil {
		if client.IsErrConnectionFailed(err) {
			// daemon isn't responding; attempt to login client side.
			return loginClientSide(ctx, options)
		}
		return client.RegistryLoginResult{}, err
	}

	return res, nil
}

func loginClientSide(ctx context.Context, options client.RegistryLoginOptions) (client.RegistryLoginResult, error) {
	svc, err := registry.NewService(registry.ServiceOptions{})
	if err != nil {
		return client.RegistryLoginResult{}, err
	}

	auth := registrytypes.AuthConfig{
		Username:      options.Username,
		Password:      options.Password,
		ServerAddress: options.ServerAddress,
		IdentityToken: options.IdentityToken,
		RegistryToken: options.RegistryToken,
	}

	token, err := svc.Auth(ctx, &auth, command.UserAgent())
	if err != nil {
		return client.RegistryLoginResult{}, err
	}

	return client.RegistryLoginResult{
		Auth: registrytypes.AuthResponse{
			Status:        "Login Succeeded",
			IdentityToken: token,
		},
	}, nil
}
