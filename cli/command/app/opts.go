package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/image"
)

// runnerName is the executable name for starting the app
const runnerName = "run"

// installerName is the executable name for custom installation
const installerName = "install"

// uninstallerName is the executable name for custom installation
const uninstallerName = "uninstall"

// namePattern is for validating egress and app name
const namePattern = "^[a-zA-Z0-9][a-zA-Z0-9_.+-]+$"

var nameRegexp = regexp.MustCompile(namePattern)

func validateName(s string) error {
	if !nameRegexp.MatchString(s) {
		return fmt.Errorf("name %q is invalid, regexp: %q", s, namePattern)
	}
	return nil
}

// defaultAppBase is docker app's base location specified by
// DOCKER_APP_BASE environment variable defaulted to ~/.docker-app/
func defaultAppBase() string {
	if base := os.Getenv("DOCKER_APP_BASE"); base != "" {
		return filepath.Clean(base)
	}
	// default ~/.docker/app
	// ignore error and use the current working directory
	// if home directory is not available
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".docker", "app")
}

// AppOptions holds the options for the `app` subcommands
type AppOptions struct {
	// flags for install

	// path on local host
	destination string

	// path in container
	egress string

	// exit immediately after launching the app
	detach bool

	// start exported app
	launch bool

	// the following are existing flags

	// build flags
	// all `docker build` flags are supported as is by `docker install`
	// iidfile is required for the run step
	// auto generated if not provided
	imageIDFile string

	// run flags
	// only a subset of run flags are supported
	// cidfile is auto generated if not provided
	containerIDFile string

	// options
	buildOpts     *image.BuildOptions
	runOpts       *container.RunOptions
	containerOpts *container.ContainerOptions
	copyOpts      *container.CopyOptions

	// runtime
	// command line args
	_args []string

	// docker app base location, fixed once set
	_appBase string
}

func (o *AppOptions) setArgs(args []string) {
	o._args = args
}

// buildContext returns the build context for building image
func (o *AppOptions) buildContext() string {
	if len(o._args) == 0 {
		return "."
	}
	return o._args[0]
}

// runArgs returns the command line args for running the container
func (o *AppOptions) runArgs() []string {
	if len(o._args) <= 1 {
		return nil
	}
	cArgs, _ := splitAtDashDash(o._args[1:])
	return cArgs
}

// launchArgs returns the command line args for launching the app
// the args after the first "--" are considered launch args
func (o *AppOptions) launchArgs() []string {
	if len(o._args) <= 1 {
		return nil
	}
	_, hArgs := splitAtDashDash(o._args[1:])
	return hArgs
}

// isDockerAppBase returns true if the destination is under the default app base
func (o *AppOptions) isDockerAppBase() bool {
	s := filepath.Clean(o.destination)
	return strings.HasPrefix(s, o._appBase)
}

// binPath returns the bin directory under the default app base
func (o *AppOptions) binPath() string {
	return filepath.Join(o._appBase, "bin")
}

// appPath returns the app directory under the default app base
func (o *AppOptions) appPath() (string, error) {
	if len(o._args) == 0 {
		return "", errors.New("missing args")
	}
	return makeAppPath(o._appBase, o._args[0])
}

// cacheDir returns a temp cache directory under the default app base
// appBase is chosen as the parent directory to avoid issues such as:
// permission, disk space, renaming across partitions.
func (o *AppOptions) cacheDir() (string, error) {
	id := time.Now().UnixNano()
	dir := filepath.Join(o._appBase, ".cache", strconv.FormatInt(id, 16))
	err := os.MkdirAll(dir, 0o755)
	return dir, err
}

func (o *AppOptions) imageID() (string, error) {
	if id, err := os.ReadFile(o.imageIDFile); err != nil {
		return "", err
	} else {
		// TODO investigate: -q/--quiet flag causes extra LF from the docker builder
		return strings.TrimSpace(string(id)), nil
	}
}

func (o *AppOptions) containerID() (string, error) {
	if id, err := os.ReadFile(o.containerIDFile); err != nil {
		return "", err
	} else {
		return string(id), nil
	}
}

func newAppOptions() *AppOptions {
	return &AppOptions{
		_appBase: defaultAppBase(),
	}
}

func validateAppOptions(options *AppOptions) error {
	if options.destination == "" {
		return errors.New("destination is required")
	}
	if options.egress == "" {
		return errors.New("egress is required")
	}

	name := filepath.Base(options.egress)
	if err := validateName(name); err != nil {
		return fmt.Errorf("invalid egress path: %s %v", options.egress, err)
	}
	return nil
}

type removeOptions struct {
	_appBase string
}

// makeAppPath returns the app directory under the default app base
// appBase/pkg/scheme/host/path
func (o *removeOptions) makeAppPath(s string) (string, error) {
	return makeAppPath(o._appBase, s)
}

// binPath returns the bin directory under the default app base
func (o *removeOptions) binPath() string {
	return filepath.Join(o._appBase, "bin")
}

// pkgPath returns the pkg directory under the default app base
func (o *removeOptions) pkgPath() string {
	return filepath.Join(o._appBase, "pkg")
}

func newRemoveOptions() *removeOptions {
	return &removeOptions{
		_appBase: defaultAppBase(),
	}
}

// makeAppPath builds the default app path
// in the format: appBase/pkg/scheme/host/path
func makeAppPath(appBase, s string) (string, error) {
	u, err := parseURL(s)
	if err != nil {
		return "", err
	}
	if u.Path == "" {
		return "", fmt.Errorf("missing path: %v", u)
	}

	name := filepath.Base(u.Path)
	if err := validateName(name); err != nil {
		return "", fmt.Errorf("invalid path: %s %v", u.Path, err)
	}
	return filepath.Join(appBase, "pkg", u.Scheme, u.Host, u.Path), nil
}
