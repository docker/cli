package container

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	apiclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type createOptions struct {
	name      string
	platform  string
	untrusted bool
}

// NewCreateCommand creates a new cobra.Command for `docker create`
func NewCreateCommand(dockerCli command.Cli) *cobra.Command {
	var opts createOptions
	var copts *containerOptions

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] IMAGE [COMMAND] [ARG...]",
		Short: "Create a new container",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			copts.Image = args[0]
			if len(args) > 1 {
				copts.Args = args[1:]
			}
			return runCreate(dockerCli, cmd.Flags(), &opts, copts)
		},
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	flags.StringVar(&opts.name, "name", "", "Assign a name to the container")

	// Add an explicit help that doesn't have a `-h` to prevent the conflict
	// with hostname
	flags.Bool("help", false, "Print usage")

	command.AddPlatformFlag(flags, &opts.platform)
	command.AddTrustVerificationFlags(flags, &opts.untrusted, dockerCli.ContentTrustEnabled())
	copts = addFlags(flags)
	return cmd
}

func runCreate(dockerCli command.Cli, flags *pflag.FlagSet, opts *createOptions, copts *containerOptions) error {
	containerConfig, err := parse(flags, copts)
	if err != nil {
		reportError(dockerCli.Err(), "create", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	response, err := createContainer(context.Background(), dockerCli, containerConfig, opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(dockerCli.Out(), response.ID)
	return nil
}

func pullImage(ctx context.Context, dockerCli command.Cli, image string, platform string, out io.Writer) error {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}

	authConfig := command.ResolveAuthConfig(ctx, dockerCli, repoInfo.Index)
	encodedAuth, err := command.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}

	options := types.ImageCreateOptions{
		RegistryAuth: encodedAuth,
		Platform:     platform,
	}

	responseBody, err := dockerCli.Client().ImageCreate(ctx, image, options)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesStream(
		responseBody,
		out,
		dockerCli.Out().FD(),
		dockerCli.Out().IsTerminal(),
		nil)
}

type cidFile struct {
	path    string
	file    *os.File
	written bool
}

func (cid *cidFile) Close() error {
	if cid.file == nil {
		return nil
	}
	cid.file.Close()

	if cid.written {
		return nil
	}
	if err := os.Remove(cid.path); err != nil {
		return errors.Errorf("failed to remove the CID file '%s': %s \n", cid.path, err)
	}

	return nil
}

func (cid *cidFile) Write(id string) error {
	if cid.file == nil {
		return nil
	}
	if _, err := cid.file.Write([]byte(id)); err != nil {
		return errors.Errorf("Failed to write the container ID to the file: %s", err)
	}
	cid.written = true
	return nil
}

func newCIDFile(path string) (*cidFile, error) {
	if path == "" {
		return &cidFile{}, nil
	}
	if _, err := os.Stat(path); err == nil {
		return nil, errors.Errorf("Container ID file found, make sure the other container isn't running or delete %s", path)
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, errors.Errorf("Failed to create the container ID file: %s", err)
	}

	return &cidFile{path: path, file: f}, nil
}

func createContainer(ctx context.Context, dockerCli command.Cli, containerConfig *containerConfig, opts *createOptions) (*container.ContainerCreateCreatedBody, error) {
	config := containerConfig.Config
	hostConfig := containerConfig.HostConfig
	networkingConfig := containerConfig.NetworkingConfig
	stderr := dockerCli.Err()

	var (
		trustedRef reference.Canonical
		namedRef   reference.Named
	)

	containerIDFile, err := newCIDFile(hostConfig.ContainerIDFile)
	if err != nil {
		return nil, err
	}
	defer containerIDFile.Close()

	ref, err := reference.ParseAnyReference(config.Image)
	if err != nil {
		return nil, err
	}
	if named, ok := ref.(reference.Named); ok {
		namedRef = reference.TagNameOnly(named)

		if taggedRef, ok := namedRef.(reference.NamedTagged); ok && !opts.untrusted {
			var err error
			trustedRef, err = image.TrustedReference(ctx, dockerCli, taggedRef, nil)
			if err != nil {
				return nil, err
			}
			config.Image = reference.FamiliarString(trustedRef)
		}
	}

	//create the container
	response, err := dockerCli.Client().ContainerCreate(ctx, config, hostConfig, networkingConfig, opts.name)

	//if image not found try to pull it
	if err != nil {
		if apiclient.IsErrNotFound(err) && namedRef != nil {
			fmt.Fprintf(stderr, "Unable to find image '%s' locally\n", reference.FamiliarString(namedRef))

			// we don't want to write to stdout anything apart from container.ID
			if err := pullImage(ctx, dockerCli, config.Image, opts.platform, stderr); err != nil {
				return nil, err
			}
			if taggedRef, ok := namedRef.(reference.NamedTagged); ok && trustedRef != nil {
				if err := image.TagTrusted(ctx, dockerCli, trustedRef, taggedRef); err != nil {
					return nil, err
				}
			}
			// Retry
			var retryErr error
			response, retryErr = dockerCli.Client().ContainerCreate(ctx, config, hostConfig, networkingConfig, opts.name)
			if retryErr != nil {
				return nil, retryErr
			}
		} else {
			return nil, err
		}
	}

	for _, warning := range response.Warnings {
		fmt.Fprintf(stderr, "WARNING: %s\n", warning)
	}
	err = containerIDFile.Write(response.ID)
	return &response, err
}
