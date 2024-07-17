package app

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/image"
)

// runnerName is the default executable app name for starting the app
const runnerName = "run"

// installerName is the executable name for custom installation
const installerName = "install"

// uninstallerName is the executable name for custom installation
const uninstallerName = "uninstall"

// namePattern is for validating app name
const namePattern = "^[a-zA-Z0-9][a-zA-Z0-9_.+-]+$"

var nameRegexp = regexp.MustCompile(namePattern)

func validateName(s string) error {
	if !nameRegexp.MatchString(s) {
		return fmt.Errorf("name %q is invalid, regexp: %q", s, namePattern)
	}
	return nil
}

// semverPattern is for splitting semver from a context path/URL
const semverPattern = `@v?\d+(\.\d+)?(\.\d+)?$`

var semverRegexp = regexp.MustCompile(semverPattern)

func splitSemver(s string) (string, string) {
	if semverRegexp.MatchString(s) {
		idx := strings.LastIndex(s, "@")
		// unlikely otherwise ignore
		if idx == -1 {
			return s, ""
		}
		v := s[idx+1:]
		if v[0] == 'v' {
			v = v[1:]
		}
		return s[:idx], v
	}
	return s, ""
}

// defaultAppBase is docker app's base location specified by
// DOCKER_APP_BASE environment variable defaulted to ~/.docker/app/
func defaultAppBase() string {
	if base := os.Getenv("DOCKER_APP_BASE"); base != "" {
		return filepath.Clean(base)
	}

	// locate .docker/app starting from the current working directory
	// for supporting apps on a per project basis
	wd, err := os.Getwd()
	if err == nil {
		if dir, err := locateDir(wd, ".docker"); err == nil {
			return filepath.Join(dir, "app")
		}
	}

	// default ~/.docker/app
	// ignore error and use the current working directory
	// if home directory is not available
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".docker", "app")
}

type commonOptions struct {
	// command line args
	_args []string

	// docker app base location, fixed once set
	_appBase string
}

func (o *commonOptions) setArgs(args []string) {
	o._args = args
}

// buildContext returns the build context for building image
func (o *commonOptions) buildContext() string {
	if len(o._args) == 0 {
		return "."
	}
	c, _ := splitSemver(o._args[0])
	return c
}

func (o *commonOptions) buildVersion() string {
	if len(o._args) == 0 {
		return ""
	}
	_, v := splitSemver(o._args[0])
	return v
}

// appPath returns the app directory under the default app base
func (o *commonOptions) appPath() (string, error) {
	if len(o._args) == 0 {
		return "", errors.New("missing args")
	}
	return o.makeAppPath(o._args[0])
}

// binPath returns the bin directory under the default app base
func (o *commonOptions) binPath() string {
	return filepath.Join(o._appBase, "bin")
}

// pkgPath returns the pkg directory under the default app base
func (o *commonOptions) pkgPath() string {
	return filepath.Join(o._appBase, "pkg")
}

// makeAppPath builds the default app path
// in the format: appBase/pkg/scheme/host/path
func (o *commonOptions) makeAppPath(s string) (string, error) {
	u, err := parseURL(s)
	if err != nil {
		return "", err
	}
	if u.Path == "" {
		return "", fmt.Errorf("missing path: %v", u)
	}
	p := filepath.Join(o._appBase, "pkg", u.Scheme, u.Host, shortenPath(u.Path))
	if u.Fragment == "" {
		return p, nil
	}
	return fmt.Sprintf("%s#%s", p, u.Fragment), nil
}

func (o *commonOptions) makeEnvs() (map[string]string, error) {
	envs := make(map[string]string)

	// copy the current environment
	for _, v := range os.Environ() {
		kv := strings.SplitN(v, "=", 2)
		envs[kv[0]] = kv[1]
	}

	envs["DOCKER_APP_BASE"] = o._appBase
	appPath, err := o.appPath()
	if err != nil {
		return nil, err
	}
	envs["DOCKER_APP_PATH"] = appPath

	envs["VERSION"] = o.buildVersion()

	envs["HOSTOS"] = runtime.GOOS
	envs["HOSTARCH"] = runtime.GOARCH

	// user info
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	envs["USERNAME"] = u.Username
	envs["USERHOME"] = u.HomeDir
	envs["USERID"] = u.Uid
	envs["USERGID"] = u.Gid

	return envs, nil
}

// AppOptions holds the options for the `app` subcommands
type AppOptions struct {
	commonOptions

	// flags for install

	// path on local host
	destination string

	// path in container
	egress string

	// exit immediately after launching the app
	detach bool

	// start exported app
	launch bool

	// overwrite existing app
	force bool

	// app name
	name string

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
		commonOptions: commonOptions{
			_appBase: defaultAppBase(),
		},
	}
}

func validateAppOptions(options *AppOptions) error {
	if options.destination == "" {
		return errors.New("destination is required")
	}
	if options.egress == "" {
		return errors.New("egress is required")
	}

	return nil
}

type removeOptions struct {
	commonOptions
}

func newRemoveOptions() *removeOptions {
	return &removeOptions{
		commonOptions: commonOptions{
			_appBase: defaultAppBase(),
		},
	}
}
