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

func runLogin(ctx context.Context, dockerCli command.Cli, opts loginOptions) error { //nolint:gocyclo
	clnt := dockerCli.Client()
	if err := verifyloginOptions(dockerCli, &opts); err != nil {
		return err
	}
	var (
		serverAddress string
		response      registrytypes.AuthenticateOKBody
	)
	if opts.serverAddress != "" && opts.serverAddress != registry.DefaultNamespace {
		serverAddress = opts.serverAddress
	} else {
		serverAddress = registry.IndexServer
	}

	isDefaultRegistry := serverAddress == registry.IndexServer
	authConfig, err := command.GetDefaultAuthConfig(dockerCli.ConfigFile(), opts.user == "" && opts.password == "", serverAddress, isDefaultRegistry)
	if err == nil && authConfig.Username != "" && authConfig.Password != "" {
		response, err = loginWithCredStoreCreds(ctx, dockerCli, &authConfig)
	}
	if err != nil || authConfig.Username == "" || authConfig.Password == "" {
		err = command.ConfigureAuth(ctx, dockerCli, opts.user, opts.password, &authConfig, isDefaultRegistry)
		if err != nil {
			return err
		}

		response, err = clnt.RegistryLogin(ctx, authConfig)
		if err != nil && client.IsErrConnectionFailed(err) {
			// If the server isn't responding (yet) attempt to login purely client side
			response, err = loginClientSide(ctx, authConfig)
		}
		// If we (still) have an error, give up
		if err != nil {
			return err
		}
	}
	if response.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = response.IdentityToken
	}

	creds := dockerCli.ConfigFile().GetCredentialsStore(serverAddress)
	if err := creds.Store(configtypes.AuthConfig(authConfig)); err != nil {
		return errors.Errorf("Error saving credentials: %v", err)
	}

	if response.Status != "" {
		fmt.Fprintln(dockerCli.Out(), response.Status)
	}
	return nil
}

func loginWithCredStoreCreds(ctx context.Context, dockerCli command.Cli, authConfig *registrytypes.AuthConfig) (registrytypes.AuthenticateOKBody, error) {
	fmt.Fprintf(dockerCli.Out(), "Authenticating with existing credentials...\n")
	cliClient := dockerCli.Client()
	response, err := cliClient.RegistryLogin(ctx, *authConfig)
	if err != nil {
		if errdefs.IsUnauthorized(err) {
			fmt.Fprintf(dockerCli.Err(), "Stored credentials invalid or expired\n")
		} else {
			fmt.Fprintf(dockerCli.Err(), "Login did not succeed, error: %s\n", err)
		}
	}
	return response, err
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
