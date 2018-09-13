package engine

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/licenseutils"
	clitypes "github.com/docker/cli/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/licensing/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type activateOptions struct {
	licenseFile    string
	version        string
	registryPrefix string
	format         string
	image          string
	quiet          bool
	displayOnly    bool
	sockPath       string
}

// newActivateCommand creates a new `docker engine activate` command
func newActivateCommand(dockerCli command.Cli) *cobra.Command {
	var options activateOptions

	cmd := &cobra.Command{
		Use:   "activate [OPTIONS]",
		Short: "Activate Enterprise Edition",
		Long: `Activate Enterprise Edition.

With this command you may apply an existing Docker enterprise license, or
interactively download one from Docker. In the interactive exchange, you can
sign up for a new trial, or download an existing license. If you are
currently running a Community Edition engine, the daemon will be updated to
the Enterprise Edition Docker engine with additional capabilities and long
term support.

For more information about different Docker Enterprise license types visit
https://www.docker.com/licenses

For non-interactive scriptable deployments, download your license from
https://hub.docker.com/ then specify the file with the '--license' flag.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runActivate(dockerCli, options)
		},
	}

	flags := cmd.Flags()

	flags.StringVar(&options.licenseFile, "license", "", "License File")
	flags.StringVar(&options.version, "version", "", "Specify engine version (default is to use currently running version)")
	flags.StringVar(&options.registryPrefix, "registry-prefix", "docker.io/store/docker", "Override the default location where engine images are pulled")
	flags.StringVar(&options.image, "engine-image", clitypes.EnterpriseEngineImage, "Specify engine image")
	flags.StringVar(&options.format, "format", "", "Pretty-print licenses using a Go template")
	flags.BoolVar(&options.displayOnly, "display-only", false, "only display the available licenses and exit")
	flags.BoolVar(&options.quiet, "quiet", false, "Only display available licenses by ID")
	flags.StringVar(&options.sockPath, "containerd", "", "override default location of containerd endpoint")

	return cmd
}

func runActivate(cli command.Cli, options activateOptions) error {
	ctx := context.Background()
	client, err := cli.NewContainerizedEngineClient(options.sockPath)
	if err != nil {
		return errors.Wrap(err, "unable to access local containerd")
	}
	defer client.Close()

	authConfig, err := getRegistryAuth(cli, options.registryPrefix)
	if err != nil {
		return err
	}

	var license *model.IssuedLicense

	// Lookup on hub if no license provided via params
	if options.licenseFile == "" {
		if license, err = getLicenses(ctx, authConfig, cli, options); err != nil {
			return err
		}
		if options.displayOnly {
			return nil
		}
	} else {
		if license, err = licenseutils.LoadLocalIssuedLicense(ctx, options.licenseFile); err != nil {
			return err
		}
	}
	if err = licenseutils.ApplyLicense(ctx, cli.Client(), license); err != nil {
		return err
	}

	opts := clitypes.EngineInitOptions{
		RegistryPrefix: options.registryPrefix,
		EngineImage:    options.image,
		EngineVersion:  options.version,
	}

	return client.ActivateEngine(ctx, opts, cli.Out(), authConfig,
		func(ctx context.Context) error {
			client := cli.Client()
			_, err := client.Ping(ctx)
			return err
		})
}

func getLicenses(ctx context.Context, authConfig *types.AuthConfig, cli command.Cli, options activateOptions) (*model.IssuedLicense, error) {
	user, err := licenseutils.Login(ctx, authConfig)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(cli.Out(), "Looking for existing licenses for %s...\n", user.User.Username)
	subs, err := user.GetAvailableLicenses(ctx)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return doTrialFlow(ctx, cli, user)
	}

	format := options.format
	if len(format) == 0 {
		format = formatter.TableFormatKey
	}

	updatesCtx := formatter.Context{
		Output: cli.Out(),
		Format: formatter.NewSubscriptionsFormat(format, options.quiet),
		Trunc:  false,
	}
	if err := formatter.SubscriptionsWrite(updatesCtx, subs); err != nil {
		return nil, err
	}
	if options.displayOnly {
		return nil, nil
	}
	fmt.Fprintf(cli.Out(), "Please pick a license by number: ")
	var num int
	if _, err := fmt.Fscan(cli.In(), &num); err != nil {
		return nil, errors.Wrap(err, "failed to read user input")
	}
	if num < 0 || num >= len(subs) {
		return nil, fmt.Errorf("invalid choice")
	}
	return user.GetIssuedLicense(ctx, subs[num].ID)
}

func doTrialFlow(ctx context.Context, cli command.Cli, user licenseutils.HubUser) (*model.IssuedLicense, error) {
	if !command.PromptForConfirmation(cli.In(), cli.Out(),
		"No existing licenses found, would you like to set up a new Enterprise Basic Trial license?") {
		return nil, fmt.Errorf("you must have an existing enterprise license or generate a new trial to use the Enterprise Docker Engine")
	}
	targetID := user.User.ID
	// If the user is a member of any organizations, allow trials generated against them
	if len(user.Orgs) > 0 {
		fmt.Fprintf(cli.Out(), "%d\t%s\n", 0, user.User.Username)
		for i, org := range user.Orgs {
			fmt.Fprintf(cli.Out(), "%d\t%s\n", i+1, org.Orgname)
		}
		fmt.Fprintf(cli.Out(), "Please choose an account to generate the trial in:")
		var num int
		if _, err := fmt.Fscan(cli.In(), &num); err != nil {
			return nil, errors.Wrap(err, "failed to read user input")
		}
		if num < 0 || num > len(user.Orgs) {
			return nil, fmt.Errorf("invalid choice")
		}
		if num > 0 {
			targetID = user.Orgs[num-1].ID
		}
	}
	return user.GenerateTrialLicense(ctx, targetID)
}
