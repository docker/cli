package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"
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
	pull      string // always, missing, never
	quiet     bool
}

// NewCreateCommand creates a new cobra.Command for `docker create`
func NewCreateCommand(dockerCli command.Cli) *cobra.Command {
	var options createOptions
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
			return runCreate(cmd.Context(), dockerCli, cmd.Flags(), &options, copts)
		},
		Annotations: map[string]string{
			"aliases": "docker container create, docker create",
		},
		ValidArgsFunction: completion.ImageNames(dockerCli),
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	flags.StringVar(&options.name, "name", "", "Assign a name to the container")
	flags.StringVar(&options.pull, "pull", PullImageMissing, `Pull image before creating ("`+PullImageAlways+`", "|`+PullImageMissing+`", "`+PullImageNever+`")`)
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Suppress the pull output")

	// Add an explicit help that doesn't have a `-h` to prevent the conflict
	// with hostname
	flags.Bool("help", false, "Print usage")

	command.AddPlatformFlag(flags, &options.platform)
	command.AddTrustVerificationFlags(flags, &options.untrusted, dockerCli.ContentTrustEnabled())
	copts = addFlags(flags)
	return cmd
}

func runCreate(ctx context.Context, dockerCli command.Cli, flags *pflag.FlagSet, options *createOptions, copts *containerOptions) error {
	if err := validatePullOpt(options.pull); err != nil {
		reportError(dockerCli.Err(), "create", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	proxyConfig := dockerCli.ConfigFile().ParseProxyConfig(dockerCli.Client().DaemonHost(), opts.ConvertKVStringsToMapWithNil(copts.env.GetAll()))
	newEnv := []string{}
	for k, v := range proxyConfig {
		if v == nil {
			newEnv = append(newEnv, k)
		} else {
			newEnv = append(newEnv, k+"="+*v)
		}
	}
	copts.env = *opts.NewListOptsRef(&newEnv, nil)
	containerCfg, err := parse(flags, copts, dockerCli.ServerInfo().OSType)
	if err != nil {
		reportError(dockerCli.Err(), "create", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	if err = validateAPIVersion(containerCfg, dockerCli.Client().ClientVersion()); err != nil {
		reportError(dockerCli.Err(), "create", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	id, err := createContainer(ctx, dockerCli, containerCfg, options)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(dockerCli.Out(), id)
	return nil
}

// FIXME(thaJeztah): this is the only code-path that uses APIClient.ImageCreate. Rewrite this to use the regular "pull" code (or vice-versa).
func pullImage(ctx context.Context, dockerCli command.Cli, img string, options *createOptions) error {
	encodedAuth, err := command.RetrieveAuthTokenFromImage(dockerCli.ConfigFile(), img)
	if err != nil {
		return err
	}

	responseBody, err := dockerCli.Client().ImageCreate(ctx, img, imagetypes.CreateOptions{
		RegistryAuth: encodedAuth,
		Platform:     options.platform,
	})
	if err != nil {
		return err
	}
	defer responseBody.Close()

	out := dockerCli.Err()
	if options.quiet {
		out = io.Discard
	}
	return jsonmessage.DisplayJSONMessagesToStream(responseBody, streams.NewOut(out), nil)
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
		return errors.Wrapf(err, "failed to remove the CID file '%s'", cid.path)
	}

	return nil
}

func (cid *cidFile) Write(id string) error {
	if cid.file == nil {
		return nil
	}
	if _, err := cid.file.Write([]byte(id)); err != nil {
		return errors.Wrap(err, "failed to write the container ID to the file")
	}
	cid.written = true
	return nil
}

func newCIDFile(path string) (*cidFile, error) {
	if path == "" {
		return &cidFile{}, nil
	}
	if _, err := os.Stat(path); err == nil {
		return nil, errors.Errorf("container ID file found, make sure the other container isn't running or delete %s", path)
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create the container ID file")
	}

	return &cidFile{path: path, file: f}, nil
}

//nolint:gocyclo
func createContainer(ctx context.Context, dockerCli command.Cli, containerCfg *containerConfig, options *createOptions) (containerID string, err error) {
	config := containerCfg.Config
	hostConfig := containerCfg.HostConfig
	networkingConfig := containerCfg.NetworkingConfig

	warnOnOomKillDisable(*hostConfig, dockerCli.Err())
	warnOnLocalhostDNS(*hostConfig, dockerCli.Err())

	var (
		trustedRef reference.Canonical
		namedRef   reference.Named
	)

	containerIDFile, err := newCIDFile(hostConfig.ContainerIDFile)
	if err != nil {
		return "", err
	}
	defer containerIDFile.Close()

	ref, err := reference.ParseAnyReference(config.Image)
	if err != nil {
		return "", err
	}
	if named, ok := ref.(reference.Named); ok {
		namedRef = reference.TagNameOnly(named)

		if taggedRef, ok := namedRef.(reference.NamedTagged); ok && !options.untrusted {
			var err error
			trustedRef, err = image.TrustedReference(ctx, dockerCli, taggedRef)
			if err != nil {
				return "", err
			}
			config.Image = reference.FamiliarString(trustedRef)
		}
	}

	pullAndTagImage := func() error {
		if err := pullImage(ctx, dockerCli, config.Image, options); err != nil {
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
	if options.platform != "" && versions.GreaterThanOrEqualTo(dockerCli.Client().ClientVersion(), "1.41") {
		p, err := platforms.Parse(options.platform)
		if err != nil {
			return "", errors.Wrap(errdefs.InvalidParameter(err), "error parsing specified platform")
		}
		platform = &p
	}

	if options.pull == PullImageAlways {
		if err := pullAndTagImage(); err != nil {
			return "", err
		}
	}

	hostConfig.ConsoleSize[0], hostConfig.ConsoleSize[1] = dockerCli.Out().GetTtySize()

	response, err := dockerCli.Client().ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, options.name)
	if err != nil {
		// Pull image if it does not exist locally and we have the PullImageMissing option. Default behavior.
		if errdefs.IsNotFound(err) && namedRef != nil && options.pull == PullImageMissing {
			if !options.quiet {
				// we don't want to write to stdout anything apart from container.ID
				fmt.Fprintf(dockerCli.Err(), "Unable to find image '%s' locally\n", reference.FamiliarString(namedRef))
			}

			if err := pullAndTagImage(); err != nil {
				return "", err
			}

			var retryErr error
			response, retryErr = dockerCli.Client().ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, options.name)
			if retryErr != nil {
				return "", retryErr
			}
		} else {
			return "", err
		}
	}

	for _, w := range response.Warnings {
		_, _ = fmt.Fprintf(dockerCli.Err(), "WARNING: %s\n", w)
	}
	err = containerIDFile.Write(response.ID)
	return response.ID, err
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

func validatePullOpt(val string) error {
	switch val {
	case PullImageAlways, PullImageMissing, PullImageNever, "":
		// valid option, but nothing to do yet
		return nil
	default:
		return fmt.Errorf(
			"invalid pull option: '%s': must be one of %q, %q or %q",
			val,
			PullImageAlways,
			PullImageMissing,
			PullImageNever,
		)
	}
}
