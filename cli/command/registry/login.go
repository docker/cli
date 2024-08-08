package registry

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/internal/oauth/manager"
	"github.com/docker/cli/cli/oauth"
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
		Short: "Log in to a registry",
		Long:  "Log in to a registry.\nIf no server is specified, the default is defined by the daemon.",
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

func verifyloginOptions(dockerCli command.Cli, opts *loginOptions) error {
	if opts.password != "" {
		fmt.Fprintln(dockerCli.Err(), "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
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
	if err := verifyloginOptions(dockerCli, &opts); err != nil {
		return err
	}
	var (
		serverAddress string
		response      *registrytypes.AuthenticateOKBody
	)
	if opts.serverAddress != "" &&
		opts.serverAddress != registry.DefaultNamespace &&
		opts.serverAddress != registry.DefaultRegistryHost {
		serverAddress = opts.serverAddress
	} else {
		serverAddress = registry.IndexServer
	}

	// attempt login with current (stored) credentials
	authConfig, err := command.GetDefaultAuthConfig(dockerCli.ConfigFile(), opts.user == "" && opts.password == "", serverAddress)
	if err == nil && authConfig.Username != "" && authConfig.Password != "" {
		response, err = loginWithStoredCredentials(ctx, dockerCli, authConfig)
	}

	// if we failed to authenticate with stored credentials (or didn't have stored credentials),
	// prompt the user for new credentials
	if err != nil || authConfig.Username == "" || authConfig.Password == "" {
		response, err = loginUser(ctx, dockerCli, opts, authConfig.Username, serverAddress)
		if err != nil {
			return err
		}
	}

	if response != nil && response.Status != "" {
		_, _ = fmt.Fprintln(dockerCli.Out(), response.Status)
	}
	return nil
}

func loginWithStoredCredentials(ctx context.Context, dockerCli command.Cli, authConfig registrytypes.AuthConfig) (*registrytypes.AuthenticateOKBody, error) {
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

	if err := storeCredentials(dockerCli, authConfig); err != nil {
		return nil, err
	}

	return &response, err
}

func loginUser(ctx context.Context, dockerCli command.Cli, opts loginOptions, defaultUsername, serverAddress string) (*registrytypes.AuthenticateOKBody, error) {
	// If we're logging into the index server and the user didn't provide a username or password, use the device flow
	if serverAddress == registry.IndexServer && opts.user == "" && opts.password == "" {
		return loginWithDeviceCodeFlow(ctx, dockerCli)
	} else {
		return loginWithUsernameAndPassword(ctx, dockerCli, opts, defaultUsername, serverAddress)
	}
}

func loginWithUsernameAndPassword(ctx context.Context, dockerCli command.Cli, opts loginOptions, defaultUsername, serverAddress string) (*registrytypes.AuthenticateOKBody, error) {
	// Prompt user for credentials
	authConfig, err := command.ConfigureAuth(ctx, dockerCli, opts.user, opts.password, defaultUsername, serverAddress)
	if err != nil {
		return nil, err
	}

	response, err := loginWithRegistry(ctx, dockerCli, authConfig)
	if err != nil {
		return nil, err
	}

	if response.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = response.IdentityToken
	}
	if err = storeCredentials(dockerCli, authConfig); err != nil {
		return nil, err
	}

	return &response, nil
}

func loginWithDeviceCodeFlow(ctx context.Context, dockerCli command.Cli) (*registrytypes.AuthenticateOKBody, error) {
	authConfig, refreshToken, err := getOAuthCredentials(ctx, dockerCli)
	if err != nil {
		return nil, err
	}

	response, err := loginWithRegistry(ctx, dockerCli, authConfig)
	if err != nil {
		return nil, err
	}

	authConfig.Password = oauth.ConcatTokens(authConfig.Password, refreshToken)
	if err = storeCredentials(dockerCli, authConfig); err != nil {
		return nil, err
	}

	return &response, nil
}

func getOAuthCredentials(ctx context.Context, dockerCli command.Cli) (authConfig registrytypes.AuthConfig, refreshToken string, err error) {
	tokenRes, err := manager.NewManager().LoginDevice(ctx, dockerCli.Err())
	if err != nil {
		return authConfig, "", err
	}

	return registrytypes.AuthConfig{
		Username:      tokenRes.Claims.Domain.Username,
		Password:      tokenRes.AccessToken,
		Email:         tokenRes.Claims.Domain.Email,
		ServerAddress: registry.IndexServer,
	}, tokenRes.RefreshToken, nil
}

func storeCredentials(dockerCli command.Cli, authConfig registrytypes.AuthConfig) error {
	creds := dockerCli.ConfigFile().GetCredentialsStore(authConfig.ServerAddress)
	if err := creds.Store(configtypes.AuthConfig(authConfig)); err != nil {
		return errors.Errorf("Error saving credentials: %v", err)
	}

	return nil
}

func loginWithRegistry(ctx context.Context, dockerCli command.Cli, authConfig registrytypes.AuthConfig) (registrytypes.AuthenticateOKBody, error) {
	response, err := dockerCli.Client().RegistryLogin(ctx, authConfig)
	if err != nil && client.IsErrConnectionFailed(err) {
		// If the server isn't responding (yet) attempt to login purely client side
		response, err = loginClientSide(ctx, authConfig)
	}
	// If we (still) have an error, give up
	if err != nil {
		return registrytypes.AuthenticateOKBody{}, err
	}

	return response, nil
}

func loginClientSide(ctx context.Context, auth registrytypes.AuthConfig) (registrytypes.AuthenticateOKBody, error) {
	svc, err := registry.NewService(registry.ServiceOptions{})
	if err != nil {
		return registrytypes.AuthenticateOKBody{}, err
	}

	status, token, err := svc.Auth(ctx, &auth, command.UserAgent())

	return registrytypes.AuthenticateOKBody{
		Status:        status,
		IdentityToken: token,
	}, err
}
