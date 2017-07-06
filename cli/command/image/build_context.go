package image

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/urlutil"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type buildInput struct {
	buildCtx       io.ReadCloser
	dockerfileCtx  io.ReadCloser
	dockerfilePath string
	contextDir     string
	cleanups       []func()
}

func (bi *buildInput) cleanup() {
	for _, fnc := range bi.cleanups {
		fnc()
	}
}

func (bi *buildInput) addCleanup(fnc func()) {
	bi.cleanups = append([]func(){fnc}, bi.cleanups...)
}

func setupContextAndDockerfile(dockerCli command.Cli, buildBuffer *buildOutputBuffer, options buildOptions) (*buildInput, error) {
	result := &buildInput{}

	if options.dockerfileFromStdin() {
		if options.contextFromStdin() {
			return result, errors.New("invalid argument: can't use stdin for both build context and dockerfile")
		}
		result.dockerfileCtx = dockerCli.In()
	}

	var err error
	specifiedContext := options.context
	switch {
	case options.contextFromStdin():
		// buildCtx is tar archive. if stdin was dockerfile then it is wrapped
		result.buildCtx, result.dockerfilePath, err = build.GetContextFromReader(dockerCli.In(), options.dockerfileName)
	case isLocalDir(specifiedContext):
		result.contextDir, result.dockerfilePath, err = build.GetContextFromLocalDir(specifiedContext, options.dockerfileName)
	case urlutil.IsGitURL(specifiedContext):
		result.contextDir, result.dockerfilePath, err = build.GetContextFromGitURL(specifiedContext, options.dockerfileName)
		if result.contextDir != "" {
			result.addCleanup(func() { os.RemoveAll(result.contextDir) })
		}
	case urlutil.IsURL(specifiedContext):
		result.buildCtx, result.dockerfilePath, err = build.GetContextFromURL(buildBuffer.progress, specifiedContext, options.dockerfileName)
	default:
		return result, errors.Errorf("unable to prepare context: path %q not found", specifiedContext)
	}

	if err != nil {
		buildBuffer.PrintProgressBuffer(dockerCli.Err())
		return result, errors.Errorf("unable to prepare context: %s", err)
	}

	if options.stream {
		return createStreamBuildInput(result, specifiedContext)
	}

	// Context is a git url, or a local dir
	if result.buildCtx == nil {
		result.buildCtx, err = createBuildContextFromLocalDir(result.contextDir, result.dockerfilePath, options.dockerfileFromStdin())
		if err != nil {
			return result, err
		}
	}

	if result.dockerfileCtx != nil {
		result.buildCtx, result.dockerfilePath, err = build.AddDockerfileToBuildContext(result.dockerfileCtx, result.buildCtx)
		if err != nil {
			return result, err
		}
	}

	return result, err
}

func createBuildContextFromLocalDir(contextDir string, dockerfilePath string, dockerfileFromStdin bool) (io.ReadCloser, error) {
	excludes, err := build.ReadDockerignore(contextDir)
	if err != nil {
		return nil, err
	}

	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		return nil, errors.Errorf("error checking context: '%s'.", err)
	}

	// And canonicalize dockerfile name to a platform-independent one
	dockerfilePath, err = archive.CanonicalTarNameForPath(dockerfilePath)
	if err != nil {
		return nil, errors.Errorf("cannot canonicalize dockerfile path %s: %v", dockerfilePath, err)
	}

	excludes = build.TrimBuildFilesFromExcludes(excludes, dockerfilePath, dockerfileFromStdin)
	return archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.IDPair{UID: 0, GID: 0},
	})
}

func createStreamBuildInput(result *buildInput, context string) (*buildInput, error) {
	if result.buildCtx != nil {
		return result, errors.Errorf("stream is not supported for context: %s", context)
	}

	var err error
	if result.dockerfileCtx == nil {
		result.dockerfileCtx, err = os.Open(result.dockerfilePath)
		if err != nil {
			return result, errors.Wrapf(err, "failed to open %s", result.dockerfilePath)
		}
		result.addCleanup(func() { result.dockerfileCtx.Close() })
	}
	return result, nil
}

type translatorFunc func(reference.NamedTagged) (reference.Canonical, error)

// resolvedTag records the repository, tag, and resolved digest reference
// from a Dockerfile rewrite.
type resolvedTag struct {
	digestRef reference.Canonical
	tagRef    reference.NamedTagged
}

func updateBuildInputForContentTrust(ctx context.Context, dockerCli command.Cli, buildInput *buildInput) ([]*resolvedTag, error) {
	if !command.IsTrusted() {
		return nil, nil
	}
	translator := func(ref reference.NamedTagged) (reference.Canonical, error) {
		return TrustedReference(ctx, dockerCli, ref, nil)
	}
	// if there is a tar wrapper, the dockerfile needs to be replaced inside it
	if buildInput.buildCtx != nil {
		// Wrap the tar archive to replace the Dockerfile entry with the rewritten
		// Dockerfile which uses trusted pulls.
		buildCtx, resolvedTags := rewriteDockerfileForContentTrust(buildInput.buildCtx, buildInput.dockerfilePath, translator)
		buildInput.buildCtx = buildCtx
		return resolvedTags, nil
	}

	if buildInput.dockerfileCtx != nil {
		// if there was not archive context still do the possible replacements in Dockerfile
		newDockerfile, _, err := rewriteDockerfileFrom(buildInput.dockerfileCtx, translator)
		if err != nil {
			return nil, err
		}
		buildInput.dockerfileCtx = ioutil.NopCloser(bytes.NewBuffer(newDockerfile))
		// TODO: shouldn't this also return resolvedTags?
	}
	return nil, nil
}
