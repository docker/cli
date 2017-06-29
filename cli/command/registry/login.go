package registry

import (
	"fmt"
	"io/ioutil"

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
	passwordFile  string
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
	flags.StringVarP(&opts.passwordFile, "password-file", "", "", "Password file whose contents are the password itself")

	return cmd
}

// getPasswordFromFile reads the contents of the file, or stdin if the file is
// == "-".
//
// It also trims off the last \n in the file, if it exists. Most people don't
// have \ns in their password, and this allows stuff like echo "password" >
// foo, without having to remember to pass -n, or vi, which can be configured
// to automatically append newlines, etc.
//
// For users that do have a \n as the last character of their password, they
// need to store it as \n\n. I think this conforms to the principle of least
// surprise, but I could be wrong :)
func getPasswordFromFile(dockerCli command.Cli, name string) (string, error) {
	var err error
	var raw []byte

	if name == "-" {
		raw, err = ioutil.ReadAll(dockerCli.In())
	} else {
		raw, err = ioutil.ReadFile(name)
	}

	if err != nil {
		return "", err
	}

	contents := string(raw)
	if contents[len(contents)-1] == '\n' {
		contents = contents[:len(contents)-1]
	}

	return contents, nil
}

func runLogin(dockerCli command.Cli, opts loginOptions) error {
	ctx := context.Background()
	clnt := dockerCli.Client()

	if opts.password != "" {
		fmt.Fprintf(dockerCli.Err(), "Using --password via the CLI is insecure. Please use --password-file.\n")
		if opts.passwordFile != "" {
			return errors.Errorf("--password and --password-file are mutually exclusive")
		}
	}

	if opts.passwordFile != "" {
		contents, err := getPasswordFromFile(dockerCli, opts.passwordFile)
		if err != nil {
			return err
		}

		opts.password = contents
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
	if err := dockerCli.CredentialsStore(serverAddress).Store(authConfig); err != nil {
		return errors.Errorf("Error saving credentials: %v", err)
	}

	if response.Status != "" {
		fmt.Fprintln(dockerCli.Out(), response.Status)
	}
	return nil
}
