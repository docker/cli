package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/containerd/containerd/platforms"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/opts"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/versions"
	apiclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Pull constants
const (
	PullImageAlways  = "always"
	PullImageMissing = "missing" // Default (matches previous behavior)
	PullImageNever   = "never"
)

type createOptions struct {
	name      string
	platform  string
	untrusted bool
	pull      string // alway, missing, never
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
	flags.StringVar(&opts.pull, "pull", PullImageMissing,
		`Pull image before creating ("`+PullImageAlways+`"|"`+PullImageMissing+`"|"`+PullImageNever+`")`)

	// Add an explicit help that doesn't have a `-h` to prevent the conflict
	// with hostname
	flags.Bool("help", false, "Print usage")

	command.AddPlatformFlag(flags, &opts.platform)
	command.AddTrustVerificationFlags(flags, &opts.untrusted, dockerCli.ContentTrustEnabled())
	copts = addFlags(flags)
	return cmd
}

func runCreate(dockerCli command.Cli, flags *pflag.FlagSet, options *createOptions, copts *containerOptions) error {
	proxyConfig := dockerCli.ConfigFile().ParseProxyConfig(dockerCli.Client().DaemonHost(), opts.ConvertKVStringsToMapWithNil(copts.env.GetAll()))
	newEnv := []string{}
	for k, v := range proxyConfig {
		if v == nil {
			newEnv = append(newEnv, k)
		} else {
			newEnv = append(newEnv, fmt.Sprintf("%s=%s", k, *v))
		}
	}
	copts.env = *opts.NewListOptsRef(&newEnv, nil)
	containerConfig, err := parse(flags, copts, dockerCli.ServerInfo().OSType)
	if err != nil {
		reportError(dockerCli.Err(), "create", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	if err = validateAPIVersion(containerConfig, dockerCli.Client().ClientVersion()); err != nil {
		reportError(dockerCli.Err(), "create", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	response, err := createContainer(context.Background(), dockerCli, containerConfig, options)
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

// nolint: gocyclo
func createContainer(ctx context.Context, dockerCli command.Cli, containerConfig *containerConfig, opts *createOptions) (*container.ContainerCreateCreatedBody, error) {
	config := containerConfig.Config
	hostConfig := containerConfig.HostConfig
	networkingConfig := containerConfig.NetworkingConfig
	stderr := dockerCli.Err()

	warnOnOomKillDisable(*hostConfig, stderr)
	warnOnLocalhostDNS(*hostConfig, stderr)

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

	pullAndTagImage := func() error {
		if err := pullImage(ctx, dockerCli, config.Image, opts.platform, stderr); err != nil {
			return err
		}
		if taggedRef, ok := namedRef.(reference.NamedTagged); ok && trustedRef != nil {
			return image.TagTrusted(ctx, dockerCli, trustedRef, taggedRef)
		}
		return nil
	}

	var platform *specs.Platform
	// Engine API version 1.41 first introduced the option to specify platform on
	// create. It will produce an error if you try to set a platform on older API
	// versions, so check the API version here to maintain backwards
	// compatibility for CLI users.
	if opts.platform != "" && versions.GreaterThanOrEqualTo(dockerCli.Client().ClientVersion(), "1.41") {
		p, err := platforms.Parse(opts.platform)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing specified platform")
		}
		platform = &p
	}

	if opts.pull == PullImageAlways {
		if err := pullAndTagImage(); err != nil {
			return nil, err
		}
	}

	response, err := dockerCli.Client().ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, opts.name)
	if err != nil {
		// Pull image if it does not exist locally and we have the PullImageMissing option. Default behavior.
		if apiclient.IsErrNotFound(err) && namedRef != nil && opts.pull == PullImageMissing {
			// we don't want to write to stdout anything apart from container.ID
			fmt.Fprintf(stderr, "Unable to find image '%s' locally\n", reference.FamiliarString(namedRef))
			if err := pullAndTagImage(); err != nil {
				return nil, err
			}

			var retryErr error
			response, retryErr = dockerCli.Client().ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, opts.name)
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

func warnOnOomKillDisable(hostConfig container.HostConfig, stderr io.Writer) {
	if hostConfig.OomKillDisable != nil && *hostConfig.OomKillDisable && hostConfig.Memory == 0 {
		fmt.Fprintln(stderr, "WARNING: Disabling the OOM killer on containers without setting a '-m/--memory' limit may be dangerous.")
	}
}

// check the DNS settings passed via --dns against localhost regexp to warn if
// they are trying to set a DNS to a localhost address
func warnOnLocalhostDNS(hostConfig container.HostConfig, stderr io.Writer) {
	for _, dnsIP := range hostConfig.DNS {
		if isLocalhost(dnsIP) {
			fmt.Fprintf(stderr, "WARNING: Localhost DNS setting (--dns=%s) may fail in containers.\n", dnsIP)
			return
		}
	}
}

// IPLocalhost is a regex pattern for IPv4 or IPv6 loopback range.
const ipLocalhost = `((127\.([0-9]{1,3}\.){2}[0-9]{1,3})|(::1)$)`

var localhostIPRegexp = regexp.MustCompile(ipLocalhost)

// IsLocalhost returns true if ip matches the localhost IP regular expression.
// Used for determining if nameserver settings are being passed which are
// localhost addresses
func isLocalhost(ip string) bool {
	return localhostIPRegexp.MatchString(ip)
}
