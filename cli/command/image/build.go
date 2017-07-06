package image

import (
	"archive/tar"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/cli/opts"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	units "github.com/docker/go-units"
	"github.com/moby/buildkit/session"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type buildOptions struct {
	context        string
	dockerfileName string
	tags           opts.ListOpts
	labels         opts.ListOpts
	buildArgs      opts.ListOpts
	extraHosts     opts.ListOpts
	ulimits        *opts.UlimitOpt
	memory         opts.MemBytes
	memorySwap     opts.MemSwapBytes
	shmSize        opts.MemBytes
	cpuShares      int64
	cpuPeriod      int64
	cpuQuota       int64
	cpuSetCpus     string
	cpuSetMems     string
	cgroupParent   string
	isolation      string
	quiet          bool
	noCache        bool
	rm             bool
	forceRm        bool
	pull           bool
	cacheFrom      []string
	compress       bool
	securityOpt    []string
	networkMode    string
	squash         bool
	target         string
	imageIDFile    string
	stream         bool
	platform       string
}

// dockerfileFromStdin returns true when the user specified that the Dockerfile
// should be read from stdin instead of a file
func (o buildOptions) dockerfileFromStdin() bool {
	return o.dockerfileName == "-"
}

// contextFromStdin returns true when the user specified that the build context
// should be read from stdin
func (o buildOptions) contextFromStdin() bool {
	return o.context == "-"
}

func newBuildOptions() buildOptions {
	ulimits := make(map[string]*units.Ulimit)
	return buildOptions{
		tags:       opts.NewListOpts(validateTag),
		buildArgs:  opts.NewListOpts(opts.ValidateEnv),
		ulimits:    opts.NewUlimitOpt(&ulimits),
		labels:     opts.NewListOpts(opts.ValidateEnv),
		extraHosts: opts.NewListOpts(opts.ValidateExtraHost),
	}
}

// NewBuildCommand creates a new `docker build` command
func NewBuildCommand(dockerCli command.Cli) *cobra.Command {
	options := newBuildOptions()

	cmd := &cobra.Command{
		Use:   "build [OPTIONS] PATH | URL | -",
		Short: "Build an image from a Dockerfile",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.context = args[0]
			return runBuild(dockerCli, options)
		},
	}

	flags := cmd.Flags()

	flags.VarP(&options.tags, "tag", "t", "Name and optionally a tag in the 'name:tag' format")
	flags.Var(&options.buildArgs, "build-arg", "Set build-time variables")
	flags.Var(options.ulimits, "ulimit", "Ulimit options")
	flags.StringVarP(&options.dockerfileName, "file", "f", "", "Name of the Dockerfile (Default is 'PATH/Dockerfile')")
	flags.VarP(&options.memory, "memory", "m", "Memory limit")
	flags.Var(&options.memorySwap, "memory-swap", "Swap limit equal to memory plus swap: '-1' to enable unlimited swap")
	flags.Var(&options.shmSize, "shm-size", "Size of /dev/shm")
	flags.Int64VarP(&options.cpuShares, "cpu-shares", "c", 0, "CPU shares (relative weight)")
	flags.Int64Var(&options.cpuPeriod, "cpu-period", 0, "Limit the CPU CFS (Completely Fair Scheduler) period")
	flags.Int64Var(&options.cpuQuota, "cpu-quota", 0, "Limit the CPU CFS (Completely Fair Scheduler) quota")
	flags.StringVar(&options.cpuSetCpus, "cpuset-cpus", "", "CPUs in which to allow execution (0-3, 0,1)")
	flags.StringVar(&options.cpuSetMems, "cpuset-mems", "", "MEMs in which to allow execution (0-3, 0,1)")
	flags.StringVar(&options.cgroupParent, "cgroup-parent", "", "Optional parent cgroup for the container")
	flags.StringVar(&options.isolation, "isolation", "", "Container isolation technology")
	flags.Var(&options.labels, "label", "Set metadata for an image")
	flags.BoolVar(&options.noCache, "no-cache", false, "Do not use cache when building the image")
	flags.BoolVar(&options.rm, "rm", true, "Remove intermediate containers after a successful build")
	flags.BoolVar(&options.forceRm, "force-rm", false, "Always remove intermediate containers")
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Suppress the build output and print image ID on success")
	flags.BoolVar(&options.pull, "pull", false, "Always attempt to pull a newer version of the image")
	flags.StringSliceVar(&options.cacheFrom, "cache-from", []string{}, "Images to consider as cache sources")
	flags.BoolVar(&options.compress, "compress", false, "Compress the build context using gzip")
	flags.StringSliceVar(&options.securityOpt, "security-opt", []string{}, "Security options")
	flags.StringVar(&options.networkMode, "network", "default", "Set the networking mode for the RUN instructions during build")
	flags.SetAnnotation("network", "version", []string{"1.25"})
	flags.Var(&options.extraHosts, "add-host", "Add a custom host-to-IP mapping (host:ip)")
	flags.StringVar(&options.target, "target", "", "Set the target build stage to build.")
	flags.StringVar(&options.imageIDFile, "iidfile", "", "Write the image ID to the file")

	command.AddTrustVerificationFlags(flags)
	command.AddPlatformFlag(flags, &options.platform)

	flags.BoolVar(&options.squash, "squash", false, "Squash newly built layers into a single new layer")
	flags.SetAnnotation("squash", "experimental", nil)
	flags.SetAnnotation("squash", "version", []string{"1.25"})

	flags.BoolVar(&options.stream, "stream", false, "Stream attaches to server to negotiate build context")
	flags.SetAnnotation("stream", "experimental", nil)
	flags.SetAnnotation("stream", "version", []string{"1.31"})

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

func runBuild(dockerCli command.Cli, options buildOptions) error {
	idFile := build.NewIDFile(options.imageIDFile)
	// Avoid leaving a stale file if we eventually fail
	if err := idFile.Remove(); err != nil {
		return err
	}

	buildBuffer := newBuildBuffer(dockerCli.Out(), options.quiet)
	buildInput, err := setupContextAndDockerfile(dockerCli, buildBuffer, options)
	if err != nil {
		return err
	}
	defer buildInput.cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolvedTags, err := updateBuildInputForContentTrust(ctx, dockerCli, buildInput)
	if err != nil {
		return err
	}

	if options.compress {
		buildInput.buildCtx, err = build.Compress(buildInput.buildCtx)
		if err != nil {
			return err
		}
	}

	var contextSession *session.Session
	if options.stream {
		contextSession, err = newBuildContextSession(dockerCli, buildInput.contextDir)
		if err != nil {
			return err
		}
	}

	progressOutput := newProgressOutput(dockerCli.Out(), buildBuffer.progress)
	var body io.Reader
	var remote string
	if contextSession != nil {
		buf, err := setupContextStream(contextSession, buildInput.contextDir, progressOutput, buildBuffer.output)
		if err != nil {
			return err
		}
		buildBuffer.output = buf
		// TODO: move this to a method on bufferedWriter
		defer func() {
			select {
			case <-buf.flushed:
			case <-ctx.Done():
			}
		}()
		remote = clientSessionRemote
		body = buildInput.dockerfileCtx
	} else {
		body = setupRequestBody(progressOutput, buildInput.buildCtx)
	}

	configFile := dockerCli.ConfigFile()
	authConfigs, _ := configFile.GetAllCredentials()
	buildOptions := types.ImageBuildOptions{
		Memory:         options.memory.Value(),
		MemorySwap:     options.memorySwap.Value(),
		Tags:           options.tags.GetAll(),
		SuppressOutput: options.quiet,
		NoCache:        options.noCache,
		Remove:         options.rm,
		ForceRemove:    options.forceRm,
		PullParent:     options.pull,
		Isolation:      container.Isolation(options.isolation),
		CPUSetCPUs:     options.cpuSetCpus,
		CPUSetMems:     options.cpuSetMems,
		CPUShares:      options.cpuShares,
		CPUQuota:       options.cpuQuota,
		CPUPeriod:      options.cpuPeriod,
		CgroupParent:   options.cgroupParent,
		Dockerfile:     buildInput.dockerfilePath,
		ShmSize:        options.shmSize.Value(),
		Ulimits:        options.ulimits.GetList(),
		BuildArgs:      configFile.ParseProxyConfig(dockerCli.Client().DaemonHost(), options.buildArgs.GetAll()),
		AuthConfigs:    authConfigs,
		Labels:         opts.ConvertKVStringsToMap(options.labels.GetAll()),
		CacheFrom:      options.cacheFrom,
		SecurityOpt:    options.securityOpt,
		NetworkMode:    options.networkMode,
		Squash:         options.squash,
		ExtraHosts:     options.extraHosts.GetAll(),
		Target:         options.target,
		RemoteContext:  remote,
		Platform:       options.platform,
	}

	buildOptions.SessionID = startSession(ctx, contextSession, dockerCli.Client(), cancel)

	response, err := dockerCli.Client().ImageBuild(ctx, body, buildOptions)
	if err != nil {
		buildBuffer.PrintBuffers(dockerCli.Err())
		cancel()
		return err
	}
	defer response.Body.Close()

	imageID, err := streamResponse(dockerCli, response, buildBuffer)
	if err != nil {
		return nil
	}

	warnBuildWindowsToLinux(dockerCli.Out(), response.OSType, options.quiet)

	// Print the imageID if it's buffered (when options.quiet is set)
	buildBuffer.PrintOutputBuffer(dockerCli.Out())

	if err := idFile.Save(imageID); err != nil {
		return err
	}

	return tagTrustedImagesFromBuild(ctx, dockerCli, resolvedTags)
}

func isLocalDir(c string) bool {
	_, err := os.Stat(c)
	return err == nil
}

func setupRequestBody(progressOutput progress.Output, buildCtx io.ReadCloser) io.Reader {
	return progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")
}

func setupContextStream(contextSession *session.Session, contextDir string, progressOutput progress.Output, out io.Writer) (*bufferedWriter, error) {
	syncDone := make(chan error) // used to signal first progress reporting completed.
	// progress would also send errors but don't need it here as errors
	// are handled by session.Run() and ImageBuild()
	if err := addDirToSession(contextSession, contextDir, progressOutput, syncDone); err != nil {
		return nil, err
	}
	return newBufferedWriter(syncDone, out), nil
}

// Setup an upload progress bar
func newProgressOutput(out *command.OutStream, progressWriter io.Writer) progress.Output {
	progressOutput := streamformatter.NewProgressOutput(progressWriter)
	if !out.IsTerminal() {
		progressOutput = &lastProgressOutput{output: progressOutput}
	}
	return progressOutput
}

// validateTag checks if the given image name can be resolved.
func validateTag(rawRepo string) (string, error) {
	_, err := reference.ParseNormalizedNamed(rawRepo)
	if err != nil {
		return "", err
	}

	return rawRepo, nil
}

var dockerfileFromLinePattern = regexp.MustCompile(`(?i)^[\s]*FROM[ \f\r\t\v]+(?P<image>[^ \f\r\t\v\n#]+)`)

// rewriteDockerfileFrom rewrites the given Dockerfile by resolving images in
// "FROM <image>" instructions to a digest reference. `translator` is a
// function that takes a repository name and tag reference and returns a
// trusted digest reference.
func rewriteDockerfileFrom(dockerfile io.Reader, translator translatorFunc) (newDockerfile []byte, resolvedTags []*resolvedTag, err error) {
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
			if ref, ok := ref.(reference.NamedTagged); ok && command.IsTrusted() {
				trustedRef, err := translator(ref)
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

func rewriteDockerfileForContentTrust(buildCtx io.ReadCloser, dockerfileName string, translator translatorFunc) (io.ReadCloser, []*resolvedTag) {
	var resolvedTags []*resolvedTag
	return archive.ReplaceFileTarWrapper(buildCtx, map[string]archive.TarModifierFunc{
		dockerfileName: func(_ string, hdr *tar.Header, content io.Reader) (*tar.Header, []byte, error) {
			var newDockerfile []byte
			var err error
			newDockerfile, resolvedTags, err = rewriteDockerfileFrom(content, translator)
			hdr.Size = int64(len(newDockerfile))
			return hdr, newDockerfile, err
		},
	}), resolvedTags
}

// buildOutputBuffer manages io.Writers used to print build progress and errors
type buildOutputBuffer struct {
	output   io.Writer
	progress io.Writer
}

func newBuildBuffer(out io.Writer, quiet bool) *buildOutputBuffer {
	progress := out
	if quiet {
		out = new(bytes.Buffer)
		progress = new(bytes.Buffer)
	}
	return &buildOutputBuffer{output: out, progress: progress}
}

// PrintBuffers to the writer
func (bb *buildOutputBuffer) PrintBuffers(writer io.Writer) {
	bb.PrintProgressBuffer(writer)
	bb.PrintOutputBuffer(writer)
}

// PrintOutputBuffer to the writer if output is a buffer
func (bb *buildOutputBuffer) PrintOutputBuffer(writer io.Writer) {
	if buffer, ok := bb.output.(*bytes.Buffer); ok {
		fmt.Fprintf(writer, buffer.String())
	}
}

// PrintProgressBuffer to the writer with a trailing newline if progress is a buffer
func (bb *buildOutputBuffer) PrintProgressBuffer(writer io.Writer) {
	if buffer, ok := bb.progress.(*bytes.Buffer); ok {
		fmt.Fprintln(writer, buffer.String())
	}
}

func streamResponse(dockerCli command.Cli, response types.ImageBuildResponse, buffer *buildOutputBuffer) (string, error) {
	var imageID string
	aux := func(auxJSON *json.RawMessage) {
		var result types.BuildResult
		if err := json.Unmarshal(*auxJSON, &result); err != nil {
			fmt.Fprintf(dockerCli.Err(), "Failed to parse aux message: %s", err)
		} else {
			imageID = result.ID
		}
	}

	err := jsonmessage.DisplayJSONMessagesStream(response.Body, buffer.output, dockerCli.Out().FD(), dockerCli.Out().IsTerminal(), aux)
	if err != nil {
		if jerr, ok := err.(*jsonmessage.JSONError); ok {
			// If no error code is set, default to 1
			if jerr.Code == 0 {
				jerr.Code = 1
			}
			buffer.PrintBuffers(dockerCli.Err())
			return "", cli.StatusError{Status: jerr.Message, StatusCode: jerr.Code}
		}
		return "", err
	}
	return imageID, nil
}

func warnBuildWindowsToLinux(out io.Writer, osType string, quiet bool) {
	// Windows: show error message about modified file permissions if the
	// daemon isn't running Windows.
	if osType != "windows" && runtime.GOOS == "windows" && !quiet {
		fmt.Fprintln(out, "SECURITY WARNING: You are building a Docker "+
			"image from Windows against a non-Windows Docker host. All files and "+
			"directories added to build context will have '-rwxr-xr-x' permissions. "+
			"It is recommended to double check and reset permissions for sensitive "+
			"files and directories.")
	}
}

func tagTrustedImagesFromBuild(ctx context.Context, dockerCli command.Cli, resolvedTags []*resolvedTag) error {
	if !command.IsTrusted() {
		return nil
	}
	// Since the build was successful, now we must tag any of the resolved
	// images from the above Dockerfile rewrite.
	for _, resolved := range resolvedTags {
		if err := TagTrusted(ctx, dockerCli, resolved.digestRef, resolved.tagRef); err != nil {
			return err
		}
	}
	return nil
}
