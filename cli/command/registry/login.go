package registry

import (
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/net/context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
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

func runLogin(dockerCli command.Cli, opts loginOptions) error {
	ctx := context.Background()
	clnt := dockerCli.Client()

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

	var (
		serverAddress string
		authServer    = command.ElectAuthServer(ctx, dockerCli)
	)
	if opts.serverAddress != "" && opts.serverAddress != registry.DefaultNamespace {
		serverAddress = opts.serverAddress
	} else {
		serverAddress = authServer
	}

	isDefaultRegistry := serverAddress == authServer

	authConfig, err := command.ConfigureAuth(dockerCli, opts.user, opts.password, serverAddress, isDefaultRegistry)
	if err != nil {
		return err
	}
	response, err := clnt.RegistryLogin(ctx, authConfig)
	if err != nil {
		return err
	}
	if response.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = response.IdentityToken
	}
	if err := dockerCli.ConfigFile().GetCredentialsStore(serverAddress).Store(authConfig); err != nil {
		return errors.Errorf("Error saving credentials: %v", err)
	}

	if response.Status != "" {
		fmt.Fprintln(dockerCli.Out(), response.Status)
	}
	return nil
}
