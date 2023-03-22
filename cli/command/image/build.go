package image

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/cli/opts"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/builder/remotecontext/urlutil"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var errStdinConflict = errors.New("invalid argument: can't use stdin for both build context and dockerfile")

type BuildOptions struct {
	Context        string
	DockerfileName string
	Tags           opts.ListOpts
	Labels         opts.ListOpts
	BuildArgs      opts.ListOpts
	ExtraHosts     opts.ListOpts
	Ulimits        *opts.UlimitOpt
	Memory         opts.MemBytes
	MemorySwap     opts.MemSwapBytes
	ShmSize        opts.MemBytes
	CpuShares      int64
	CpuPeriod      int64
	CpuQuota       int64
	CpuSetCpus     string
	CpuSetMems     string
	CgroupParent   string
	Isolation      string
	Quiet          bool
	NoCache        bool
	Rm             bool
	ForceRm        bool
	Pull           bool
	CacheFrom      []string
	Compress       bool
	SecurityOpt    []string
	NetworkMode    string
	Squash         bool
	Target         string
	ImageIDFile    string
	Platform       string
	Untrusted      bool
}

// dockerfileFromStdin returns true when the user specified that the Dockerfile
// should be read from stdin instead of a file
func (o BuildOptions) dockerfileFromStdin() bool {
	return o.DockerfileName == "-"
}

// contextFromStdin returns true when the user specified that the build context
// should be read from stdin
func (o BuildOptions) contextFromStdin() bool {
	return o.Context == "-"
}

func NewBuildOptions() BuildOptions {
	ulimits := make(map[string]*units.Ulimit)
	return BuildOptions{
		Tags:       opts.NewListOpts(validateTag),
		BuildArgs:  opts.NewListOpts(opts.ValidateEnv),
		Ulimits:    opts.NewUlimitOpt(&ulimits),
		Labels:     opts.NewListOpts(opts.ValidateLabel),
		ExtraHosts: opts.NewListOpts(opts.ValidateExtraHost),
	}
}

// NewBuildCommand creates a new `docker build` command
func NewBuildCommand(dockerCli command.Cli) *cobra.Command {
	options := NewBuildOptions()

	cmd := &cobra.Command{
		Use:   "build [OPTIONS] PATH | URL | -",
		Short: "Build an image from a Dockerfile",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Context = args[0]
			return RunBuild(dockerCli, options)
		},
		Annotations: map[string]string{
			"category-top": "4",
			"aliases":      "docker image build, docker build, docker buildx build, docker builder build",
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
	}

	flags := cmd.Flags()

	flags.VarP(&options.Tags, "tag", "t", `Name and optionally a tag in the "name:tag" format`)
	flags.Var(&options.BuildArgs, "build-arg", "Set build-time variables")
	flags.Var(options.Ulimits, "ulimit", "Ulimit options")
	flags.StringVarP(&options.DockerfileName, "file", "f", "", `Name of the Dockerfile (Default is "PATH/Dockerfile")`)
	flags.VarP(&options.Memory, "memory", "m", "Memory limit")
	flags.Var(&options.MemorySwap, "memory-swap", `Swap limit equal to memory plus swap: -1 to enable unlimited swap`)
	flags.Var(&options.ShmSize, "shm-size", `Size of "/dev/shm"`)
	flags.Int64VarP(&options.CpuShares, "cpu-shares", "c", 0, "CPU shares (relative weight)")
	flags.Int64Var(&options.CpuPeriod, "cpu-period", 0, "Limit the CPU CFS (Completely Fair Scheduler) period")
	flags.Int64Var(&options.CpuQuota, "cpu-quota", 0, "Limit the CPU CFS (Completely Fair Scheduler) quota")
	flags.StringVar(&options.CpuSetCpus, "cpuset-cpus", "", "CPUs in which to allow execution (0-3, 0,1)")
	flags.StringVar(&options.CpuSetMems, "cpuset-mems", "", "MEMs in which to allow execution (0-3, 0,1)")
	flags.StringVar(&options.CgroupParent, "cgroup-parent", "", "Optional parent cgroup for the container")
	flags.StringVar(&options.Isolation, "isolation", "", "Container isolation technology")
	flags.Var(&options.Labels, "label", "Set metadata for an image")
	flags.BoolVar(&options.NoCache, "no-cache", false, "Do not use cache when building the image")
	flags.BoolVar(&options.Rm, "rm", true, "Remove intermediate containers after a successful build")
	flags.BoolVar(&options.ForceRm, "force-rm", false, "Always remove intermediate containers")
	flags.BoolVarP(&options.Quiet, "quiet", "q", false, "Suppress the build output and print image ID on success")
	flags.BoolVar(&options.Pull, "pull", false, "Always attempt to pull a newer version of the image")
	flags.StringSliceVar(&options.CacheFrom, "cache-from", []string{}, "Images to consider as cache sources")
	flags.BoolVar(&options.Compress, "compress", false, "Compress the build context using gzip")
	flags.StringSliceVar(&options.SecurityOpt, "security-opt", []string{}, "Security options")
	flags.StringVar(&options.NetworkMode, "network", "default", "Set the networking mode for the RUN instructions during build")
	flags.SetAnnotation("network", "version", []string{"1.25"})
	flags.Var(&options.ExtraHosts, "add-host", `Add a custom host-to-IP mapping ("host:ip")`)
	flags.StringVar(&options.Target, "target", "", "Set the target build stage to build.")
	flags.StringVar(&options.ImageIDFile, "iidfile", "", "Write the image ID to the file")

	command.AddTrustVerificationFlags(flags, &options.Untrusted, dockerCli.ContentTrustEnabled())

	flags.StringVar(&options.Platform, "platform", os.Getenv("DOCKER_DEFAULT_PLATFORM"), "Set platform if server is multi-platform capable")
	flags.SetAnnotation("platform", "version", []string{"1.38"})

	flags.BoolVar(&options.Squash, "squash", false, "Squash newly built layers into a single new layer")
	flags.SetAnnotation("squash", "experimental", nil)
	flags.SetAnnotation("squash", "version", []string{"1.25"})

	return cmd
}

// lastProgressOutput is the same as progress.Output except
// that it only output with the last update. It is used in
// non terminal scenarios to suppress verbose messages
type lastProgressOutput struct {
	output progress.Output
}

// WriteProgress formats progress information from a ProgressReader.
func (out *lastProgressOutput) WriteProgress(prog progress.Progress) error {
	if !prog.LastUpdate {
		return nil
	}

	return out.output.WriteProgress(prog)
}

//nolint:gocyclo
func RunBuild(dockerCli command.Cli, options BuildOptions) error {
	var (
		err           error
		buildCtx      io.ReadCloser
		dockerfileCtx io.ReadCloser
		contextDir    string
		tempDir       string
		relDockerfile string
		progBuff      io.Writer
		buildBuff     io.Writer
		remote        string
	)

	if options.dockerfileFromStdin() {
		if options.contextFromStdin() {
			return errStdinConflict
		}
		dockerfileCtx = dockerCli.In()
	}

	specifiedContext := options.Context
	progBuff = dockerCli.Out()
	buildBuff = dockerCli.Out()
	if options.Quiet {
		progBuff = bytes.NewBuffer(nil)
		buildBuff = bytes.NewBuffer(nil)
	}
	if options.ImageIDFile != "" {
		// Avoid leaving a stale file if we eventually fail
		if err := os.Remove(options.ImageIDFile); err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "Removing image ID file")
		}
	}

	switch {
	case options.contextFromStdin():
		// buildCtx is tar archive. if stdin was dockerfile then it is wrapped
		buildCtx, relDockerfile, err = build.GetContextFromReader(dockerCli.In(), options.DockerfileName)
	case isLocalDir(specifiedContext):
		contextDir, relDockerfile, err = build.GetContextFromLocalDir(specifiedContext, options.DockerfileName)
		if err == nil && strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
			// Dockerfile is outside of build-context; read the Dockerfile and pass it as dockerfileCtx
			dockerfileCtx, err = os.Open(options.DockerfileName)
			if err != nil {
				return errors.Errorf("unable to open Dockerfile: %v", err)
			}
			defer dockerfileCtx.Close()
		}
	case urlutil.IsGitURL(specifiedContext):
		tempDir, relDockerfile, err = build.GetContextFromGitURL(specifiedContext, options.DockerfileName)
	case urlutil.IsURL(specifiedContext):
		buildCtx, relDockerfile, err = build.GetContextFromURL(progBuff, specifiedContext, options.DockerfileName)
	default:
		return errors.Errorf("unable to prepare context: path %q not found", specifiedContext)
	}

	if err != nil {
		if options.Quiet && urlutil.IsURL(specifiedContext) {
			fmt.Fprintln(dockerCli.Err(), progBuff)
		}
		return errors.Errorf("unable to prepare context: %s", err)
	}

	if tempDir != "" {
		defer os.RemoveAll(tempDir)
		contextDir = tempDir
	}

	// read from a directory into tar archive
	if buildCtx == nil {
		excludes, err := build.ReadDockerignore(contextDir)
		if err != nil {
			return err
		}

		if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
			return errors.Wrap(err, "error checking context")
		}

		// And canonicalize dockerfile name to a platform-independent one
		relDockerfile = filepath.ToSlash(relDockerfile)

		excludes = build.TrimBuildFilesFromExcludes(excludes, relDockerfile, options.dockerfileFromStdin())
		buildCtx, err = archive.TarWithOptions(contextDir, &archive.TarOptions{
			ExcludePatterns: excludes,
			ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
		})
		if err != nil {
			return err
		}
	}

	// replace Dockerfile if it was added from stdin or a file outside the build-context, and there is archive context
	if dockerfileCtx != nil && buildCtx != nil {
		buildCtx, relDockerfile, err = build.AddDockerfileToBuildContext(dockerfileCtx, buildCtx)
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var resolvedTags []*resolvedTag
	if !options.Untrusted {
		translator := func(ctx context.Context, ref reference.NamedTagged) (reference.Canonical, error) {
			return TrustedReference(ctx, dockerCli, ref, nil)
		}
		// if there is a tar wrapper, the dockerfile needs to be replaced inside it
		if buildCtx != nil {
			// Wrap the tar archive to replace the Dockerfile entry with the rewritten
			// Dockerfile which uses trusted pulls.
			buildCtx = replaceDockerfileForContentTrust(ctx, buildCtx, relDockerfile, translator, &resolvedTags)
		} else if dockerfileCtx != nil {
			// if there was not archive context still do the possible replacements in Dockerfile
			newDockerfile, _, err := rewriteDockerfileFromForContentTrust(ctx, dockerfileCtx, translator)
			if err != nil {
				return err
			}
			dockerfileCtx = io.NopCloser(bytes.NewBuffer(newDockerfile))
		}
	}

	if options.Compress {
		buildCtx, err = build.Compress(buildCtx)
		if err != nil {
			return err
		}
	}

	// Setup an upload progress bar
	progressOutput := streamformatter.NewProgressOutput(progBuff)
	if !dockerCli.Out().IsTerminal() {
		progressOutput = &lastProgressOutput{output: progressOutput}
	}

	// if up to this point nothing has set the context then we must have another
	// way for sending it(streaming) and set the context to the Dockerfile
	if dockerfileCtx != nil && buildCtx == nil {
		buildCtx = dockerfileCtx
	}

	var body io.Reader
	if buildCtx != nil {
		body = progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")
	}

	configFile := dockerCli.ConfigFile()
	creds, _ := configFile.GetAllCredentials()
	authConfigs := make(map[string]types.AuthConfig, len(creds))
	for k, auth := range creds {
		authConfigs[k] = types.AuthConfig(auth)
	}
	buildOptions := imageBuildOptions(dockerCli, options)
	buildOptions.Version = types.BuilderV1
	buildOptions.Dockerfile = relDockerfile
	buildOptions.AuthConfigs = authConfigs
	buildOptions.RemoteContext = remote

	response, err := dockerCli.Client().ImageBuild(ctx, body, buildOptions)
	if err != nil {
		if options.Quiet {
			fmt.Fprintf(dockerCli.Err(), "%s", progBuff)
		}
		cancel()
		return err
	}
	defer response.Body.Close()

	imageID := ""
	aux := func(msg jsonmessage.JSONMessage) {
		var result types.BuildResult
		if err := json.Unmarshal(*msg.Aux, &result); err != nil {
			fmt.Fprintf(dockerCli.Err(), "Failed to parse aux message: %s", err)
		} else {
			imageID = result.ID
		}
	}

	err = jsonmessage.DisplayJSONMessagesStream(response.Body, buildBuff, dockerCli.Out().FD(), dockerCli.Out().IsTerminal(), aux)
	if err != nil {
		if jerr, ok := err.(*jsonmessage.JSONError); ok {
			// If no error code is set, default to 1
			if jerr.Code == 0 {
				jerr.Code = 1
			}
			if options.Quiet {
				fmt.Fprintf(dockerCli.Err(), "%s%s", progBuff, buildBuff)
			}
			return cli.StatusError{Status: jerr.Message, StatusCode: jerr.Code}
		}
		return err
	}

	// Windows: show error message about modified file permissions if the
	// daemon isn't running Windows.
	if response.OSType != "windows" && runtime.GOOS == "windows" && !options.Quiet {
		fmt.Fprintln(dockerCli.Out(), "SECURITY WARNING: You are building a Docker "+
			"image from Windows against a non-Windows Docker host. All files and "+
			"directories added to build context will have '-rwxr-xr-x' permissions. "+
			"It is recommended to double check and reset permissions for sensitive "+
			"files and directories.")
	}

	// Everything worked so if -q was provided the output from the daemon
	// should be just the image ID and we'll print that to stdout.
	if options.Quiet {
		imageID = fmt.Sprintf("%s", buildBuff)
		_, _ = fmt.Fprint(dockerCli.Out(), imageID)
	}

	if options.ImageIDFile != "" {
		if imageID == "" {
			return errors.Errorf("Server did not provide an image ID. Cannot write %s", options.ImageIDFile)
		}
		if err := os.WriteFile(options.ImageIDFile, []byte(imageID), 0o666); err != nil {
			return err
		}
	}
	if !options.Untrusted {
		// Since the build was successful, now we must tag any of the resolved
		// images from the above Dockerfile rewrite.
		for _, resolved := range resolvedTags {
			if err := TagTrusted(ctx, dockerCli, resolved.digestRef, resolved.tagRef); err != nil {
				return err
			}
		}
	}

	return nil
}

func isLocalDir(c string) bool {
	_, err := os.Stat(c)
	return err == nil
}

type translatorFunc func(context.Context, reference.NamedTagged) (reference.Canonical, error)

// validateTag checks if the given image name can be resolved.
func validateTag(rawRepo string) (string, error) {
	_, err := reference.ParseNormalizedNamed(rawRepo)
	if err != nil {
		return "", err
	}

	return rawRepo, nil
}

var dockerfileFromLinePattern = regexp.MustCompile(`(?i)^[\s]*FROM[ \f\r\t\v]+(?P<image>[^ \f\r\t\v\n#]+)`)

// resolvedTag records the repository, tag, and resolved digest reference
// from a Dockerfile rewrite.
type resolvedTag struct {
	digestRef reference.Canonical
	tagRef    reference.NamedTagged
}

// rewriteDockerfileFromForContentTrust rewrites the given Dockerfile by resolving images in
// "FROM <image>" instructions to a digest reference. `translator` is a
// function that takes a repository name and tag reference and returns a
// trusted digest reference.
// This should be called *only* when content trust is enabled
func rewriteDockerfileFromForContentTrust(ctx context.Context, dockerfile io.Reader, translator translatorFunc) (newDockerfile []byte, resolvedTags []*resolvedTag, err error) {
	scanner := bufio.NewScanner(dockerfile)
	buf := bytes.NewBuffer(nil)

	// Scan the lines of the Dockerfile, looking for a "FROM" line.
	for scanner.Scan() {
		line := scanner.Text()

		matches := dockerfileFromLinePattern.FindStringSubmatch(line)
		if matches != nil && matches[1] != api.NoBaseImageSpecifier {
			// Replace the line with a resolved "FROM repo@digest"
			var ref reference.Named
			ref, err = reference.ParseNormalizedNamed(matches[1])
			if err != nil {
				return nil, nil, err
			}
			ref = reference.TagNameOnly(ref)
			if ref, ok := ref.(reference.NamedTagged); ok {
				trustedRef, err := translator(ctx, ref)
				if err != nil {
					return nil, nil, err
				}

				line = dockerfileFromLinePattern.ReplaceAllLiteralString(line, fmt.Sprintf("FROM %s", reference.FamiliarString(trustedRef)))
				resolvedTags = append(resolvedTags, &resolvedTag{
					digestRef: trustedRef,
					tagRef:    ref,
				})
			}
		}

		_, err := fmt.Fprintln(buf, line)
		if err != nil {
			return nil, nil, err
		}
	}

	return buf.Bytes(), resolvedTags, scanner.Err()
}

// replaceDockerfileForContentTrust wraps the given input tar archive stream and
// uses the translator to replace the Dockerfile which uses a trusted reference.
// Returns a new tar archive stream with the replaced Dockerfile.
func replaceDockerfileForContentTrust(ctx context.Context, inputTarStream io.ReadCloser, dockerfileName string, translator translatorFunc, resolvedTags *[]*resolvedTag) io.ReadCloser {
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		tarReader := tar.NewReader(inputTarStream)
		tarWriter := tar.NewWriter(pipeWriter)

		defer inputTarStream.Close()

		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				// Signals end of archive.
				tarWriter.Close()
				pipeWriter.Close()
				return
			}
			if err != nil {
				pipeWriter.CloseWithError(err)
				return
			}

			content := io.Reader(tarReader)
			if hdr.Name == dockerfileName {
				// This entry is the Dockerfile. Since the tar archive was
				// generated from a directory on the local filesystem, the
				// Dockerfile will only appear once in the archive.
				var newDockerfile []byte
				newDockerfile, *resolvedTags, err = rewriteDockerfileFromForContentTrust(ctx, content, translator)
				if err != nil {
					pipeWriter.CloseWithError(err)
					return
				}
				hdr.Size = int64(len(newDockerfile))
				content = bytes.NewBuffer(newDockerfile)
			}

			if err := tarWriter.WriteHeader(hdr); err != nil {
				pipeWriter.CloseWithError(err)
				return
			}

			if _, err := io.Copy(tarWriter, content); err != nil {
				pipeWriter.CloseWithError(err)
				return
			}
		}
	}()

	return pipeReader
}

func imageBuildOptions(dockerCli command.Cli, options BuildOptions) types.ImageBuildOptions {
	configFile := dockerCli.ConfigFile()
	return types.ImageBuildOptions{
		Memory:         options.Memory.Value(),
		MemorySwap:     options.MemorySwap.Value(),
		Tags:           options.Tags.GetAll(),
		SuppressOutput: options.Quiet,
		NoCache:        options.NoCache,
		Remove:         options.Rm,
		ForceRemove:    options.ForceRm,
		PullParent:     options.Pull,
		Isolation:      container.Isolation(options.Isolation),
		CPUSetCPUs:     options.CpuSetCpus,
		CPUSetMems:     options.CpuSetMems,
		CPUShares:      options.CpuShares,
		CPUQuota:       options.CpuQuota,
		CPUPeriod:      options.CpuPeriod,
		CgroupParent:   options.CgroupParent,
		ShmSize:        options.ShmSize.Value(),
		Ulimits:        options.Ulimits.GetList(),
		BuildArgs:      configFile.ParseProxyConfig(dockerCli.Client().DaemonHost(), opts.ConvertKVStringsToMapWithNil(options.BuildArgs.GetAll())),
		Labels:         opts.ConvertKVStringsToMap(options.Labels.GetAll()),
		CacheFrom:      options.CacheFrom,
		SecurityOpt:    options.SecurityOpt,
		NetworkMode:    options.NetworkMode,
		Squash:         options.Squash,
		ExtraHosts:     options.ExtraHosts.GetAll(),
		Target:         options.Target,
		Platform:       options.Platform,
	}
}
