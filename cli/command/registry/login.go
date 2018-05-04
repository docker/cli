package registry

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const unencryptedWarning = `WARNING! Your password will be stored unencrypted in %s.
Configure a credential helper to remove this warning. See
https://docs.docker.com/engine/reference/commandline/login/#credentials-store
`

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
		Short: "Log in to a Docker registry",
		Long:  "Log in to a Docker registry.\nIf no server is specified, the default is defined by the daemon.",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.serverAddress = args[0]
			}
			return runLogin(dockerCli, opts)
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.user, "username", "u", "", "Username")
	flags.StringVarP(&opts.password, "password", "p", "", "Password")
	flags.BoolVarP(&opts.passwordStdin, "password-stdin", "", false, "Take the password from stdin")

	return cmd
}

// displayUnencryptedWarning warns the user when using an insecure credential storage.
// After a deprecation period, user will get prompted if stdin and stderr are a terminal.
// Otherwise, we'll assume they want it (sadly), because people may have been scripting
// insecure logins and we don't want to break them. Maybe they'll see the warning in their
// logs and fix things.
func displayUnencryptedWarning(dockerCli command.Streams, filename string) error {
	_, err := fmt.Fprintln(dockerCli.Err(), fmt.Sprintf(unencryptedWarning, filename))

	return err
}

type isFileStore interface {
	IsFileStore() bool
	GetFilename() string
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

		contents, err := ioutil.ReadAll(dockerCli.In())
		if err != nil {
			return err
		}

		opts.password = strings.TrimSuffix(string(contents), "\n")
		opts.password = strings.TrimSuffix(opts.password, "\r")
	}
	return nil
}

func runLogin(dockerCli command.Cli, opts loginOptions) error { //nolint: gocyclo
	ctx := context.Background()
	clnt := dockerCli.Client()
	if err := verifyloginOptions(dockerCli, &opts); err != nil {
		return err
	}
	var (
		serverAddress string
		authServer    = command.ElectAuthServer(ctx, dockerCli)
	)
	if opts.serverAddress != "" && opts.serverAddress != registry.DefaultNamespace {
		serverAddress = opts.serverAddress
	} else {
		serverAddress = authServer
	}

	var err error
	var authConfig *types.AuthConfig
	var response registrytypes.AuthenticateOKBody
	isDefaultRegistry := serverAddress == authServer
	authConfig, err = command.GetDefaultAuthConfig(dockerCli, opts.user == "" && opts.password == "", serverAddress, isDefaultRegistry)
	if err == nil && authConfig.Username != "" && authConfig.Password != "" {
		response, err = loginWithCredStoreCreds(ctx, dockerCli, authConfig)
	}
	if err != nil || authConfig.Username == "" || authConfig.Password == "" {
		err = command.ConfigureAuth(dockerCli, opts.user, opts.password, authConfig, isDefaultRegistry)
		if err != nil {
			return err
		}

		response, err = clnt.RegistryLogin(ctx, *authConfig)
		if err != nil {
			return err
		}
	}
	if response.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = response.IdentityToken
	}

	creds := dockerCli.ConfigFile().GetCredentialsStore(serverAddress)

	store, isDefault := creds.(isFileStore)
	if isDefault {
		err = displayUnencryptedWarning(dockerCli, store.GetFilename())
		if err != nil {
			return err
		}
	}

	if err := creds.Store(*authConfig); err != nil {
		return errors.Errorf("Error saving credentials: %v", err)
	}

	if response.Status != "" {
		fmt.Fprintln(dockerCli.Out(), response.Status)
	}
	return nil
}

func loginWithCredStoreCreds(ctx context.Context, dockerCli command.Cli, authConfig *types.AuthConfig) (registrytypes.AuthenticateOKBody, error) {
	fmt.Fprintf(dockerCli.Out(), "Authenticating with existing credentials...\n")
	cliClient := dockerCli.Client()
	response, err := cliClient.RegistryLogin(ctx, *authConfig)
	if err != nil {
		if client.IsErrUnauthorized(err) {
			fmt.Fprintf(dockerCli.Err(), "Stored credentials invalid or expired\n")
		} else {
			fmt.Fprintf(dockerCli.Err(), "Login did not succeed, error: %s\n", err)
		}
	}
	return response, err
}
