package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/docker/errdefs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewInstallCommand creates a new `docker app install` command
func NewInstallCommand(dockerCli command.Cli) *cobra.Command {
	var options *AppOptions

	cmd := &cobra.Command{
		Use:   "install [OPTIONS] URL [COMMAND] [ARG...]",
		Short: "Install app from URL",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.setArgs(args)
			adapter := newDockerCliAdapter(dockerCli)
			return installApp(cmd.Context(), adapter, cmd.Flags(), options)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	options = addInstallFlags(flags, defaultAppBase(), dockerCli.ContentTrustEnabled())

	return cmd
}

func addInstallFlags(flags *pflag.FlagSet, dest string, trust bool) *AppOptions {
	options := newAppOptions()

	bflags := pflag.NewFlagSet("build", pflag.ContinueOnError)
	rflags := pflag.NewFlagSet("run", pflag.ContinueOnError)
	eflags := pflag.NewFlagSet("copy", pflag.ContinueOnError)

	id := time.Now().UnixNano()
	imageIDFile := filepath.Join(os.TempDir(), fmt.Sprintf("docker-app-%d.iid", id))
	containerIDFile := filepath.Join(os.TempDir(), fmt.Sprintf("docker-app-%d.cid", id))

	// install supported flags
	flags.StringVar(&options.egress, "egress", "/egress", "Set container path to export")
	flags.StringVar(&options.destination, "destination", dest, "Set local host path for app")
	flags.BoolVar(&options.launch, "launch", false, "Start app after installation")
	flags.BoolVarP(&options.detach, "detach", "d", false, "Do not wait for app to finish")
	flags.BoolVar(&options.force, "force", false, "Force install even if the app exists")
	flags.StringVar(&options.name, "name", "", "App name")

	// build/run flags
	flags.StringVar(&options.imageIDFile, "iidfile", imageIDFile, "Write the image ID to the file")
	flags.StringVar(&options.containerIDFile, "cidfile", containerIDFile, "Write the container ID to the file")

	flags.Lookup("iidfile").DefValue = "auto"
	flags.Lookup("cidfile").DefValue = "auto"

	options.buildOpts = image.AddBuildFlags(bflags, trust)
	options.runOpts = container.AddRunFlags(rflags, trust)
	options.containerOpts = container.AddFlags(rflags)
	options.copyOpts = container.AddCopyFlags(eflags)

	match := func(n string, names []string) bool {
		for _, name := range names {
			if n == name {
				return true
			}
		}
		return false
	}
	include := func(flags, fs *pflag.FlagSet, names []string) {
		fs.VisitAll(func(flag *pflag.Flag) {
			if match(flag.Name, names) {
				flags.AddFlag(flag)
			}
		})
	}
	exclude := func(flags, fs *pflag.FlagSet, names []string) {
		fs.VisitAll(func(flag *pflag.Flag) {
			if !match(flag.Name, names) {
				flags.AddFlag(flag)
			}
		})
	}

	// `docker build` flags
	// all build flags are supported as is except for "iidfile"
	// install intercepts the iidfile flag
	exclude(flags, bflags, []string{"iidfile"})
	// `docker run` flags
	// we could add more if necessary except for the following
	// that are conflicting data types or duplicates of the build flags:
	// "platform", "pull", "rm", "quiet",
	// "add-host", "cgroup-parent", "cpu-period", "cpu-quota", "cpu-shares", "cpuset-cpus", "cpuset-mems",
	// "disable-content-trust", "isolation", "label", "memory", "memory-swap",
	// "network", "security-opt", "shm-size", "tty", "ulimit"
	include(flags, rflags, []string{"entrypoint", "env", "env-file", "privileged", "volume", "workdir"})
	// `docker cp` flags
	include(flags, eflags, []string{"archive", "follow-link"})

	return options
}

func setBuildArgs(options *AppOptions) error {
	bopts := options.buildOpts
	if bopts == nil {
		return errors.New("build options not set")
	}

	set := func(n, v string) {
		bopts.SetBuildArg(n + "=" + v)
	}

	set("DOCKER_APP_BASE", options._appBase)
	appPath, err := options.appPath()
	if err != nil {
		return err
	}
	set("DOCKER_APP_PATH", appPath)

	set("HOSTOS", runtime.GOOS)
	set("HOSTARCH", runtime.GOARCH)

	version := options.buildVersion()
	if version != "" {
		set("VERSION", version)
	}

	// user info
	u, err := user.Current()
	if err != nil {
		return err
	}
	set("USERNAME", u.Username)
	set("USERHOME", u.HomeDir)
	set("USERID", u.Uid)
	set("USERGID", u.Gid)

	return nil
}

func installApp(ctx context.Context, adapter cliAdapter, flags *pflag.FlagSet, options *AppOptions) error {
	if err := validateAppOptions(options); err != nil {
		return err
	}

	dir, err := runInstall(ctx, adapter, flags, options)
	if err != nil {
		return err
	}

	bin, err := runPostInstall(ctx, adapter, dir, options)
	if err != nil {
		return err
	}

	// if launch is true, run the app
	// only for single file or multi file with the run file
	if options.launch && bin != "" {
		return launch(bin, options)
	}
	return nil
}

func setDefaultEnv() {
	if os.Getenv("DOCKER_BUILDKIT") == "" {
		os.Setenv("DOCKER_BUILDKIT", "1")
	}

	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	if os.Getenv("DOCKER_DEFAULT_PLATFORM") == "" {
		os.Setenv("DOCKER_DEFAULT_PLATFORM", platform)
	}
}

// runInstall calls the build, run, and cp commands
func runInstall(ctx context.Context, dockerCli cliAdapter, flags *pflag.FlagSet, options *AppOptions) (string, error) {
	setDefaultEnv()

	if err := setBuildArgs(options); err != nil {
		return "", err
	}

	iid, err := buildImage(ctx, dockerCli, options)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(dockerCli.Out(), "Image ID: %s\n", iid)

	cid, err := runContainer(ctx, dockerCli, iid, flags, options)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(dockerCli.Out(), "Container ID: %s\n", cid)

	dest, err := copyFiles(ctx, dockerCli, cid, options)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(dockerCli.Out(), "App copied to %s\n", dest)

	return dest, nil
}

func buildImage(ctx context.Context, dockerCli cliAdapter, options *AppOptions) (string, error) {
	bopts := options.buildOpts
	bopts.SetContext(options.buildContext())
	bopts.SetImageIDFile(options.imageIDFile)
	if err := dockerCli.RunBuild(ctx, bopts); err != nil {
		return "", err
	}

	iid, err := options.imageID()
	if err != nil {
		return "", err
	}

	return iid, nil
}

func runContainer(ctx context.Context, dockerCli cliAdapter, iid string, flags *pflag.FlagSet, options *AppOptions) (string, error) {
	ropts := options.runOpts
	copts := options.containerOpts
	copts.Image = iid
	copts.Args = options.runArgs()
	copts.SetContainerIDFile(options.containerIDFile)
	if err := dockerCli.RunRun(ctx, flags, ropts, copts); err != nil {
		return "", err
	}

	cid, err := options.containerID()
	if err != nil {
		return "", err
	}

	return cid, nil
}

func copyFiles(ctx context.Context, dockerCli cliAdapter, cid string, options *AppOptions) (string, error) {
	dir, err := options.cacheDir()
	if err != nil {
		return "", err
	}

	eopts := options.copyOpts
	eopts.SetDestination(dir)
	eopts.SetSource(fmt.Sprintf("%s:%s", cid, options.egress))
	if err := dockerCli.RunCopy(ctx, eopts); err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.Base(options.egress)), nil
}

const appExistWarn = `WARNING! This will replace the existing app.
Are you sure you want to continue?`

func runPostInstall(ctx context.Context, dockerCli cliAdapter, dir string, options *AppOptions) (string, error) {
	if !options.isDockerAppBase() {
		return "", installCustom(dir, options.destination, options)
	}

	binPath := options.binPath()
	if err := os.MkdirAll(binPath, 0o755); err != nil {
		return "", err
	}

	appPath, err := options.appPath()
	if err != nil {
		return "", err
	}

	if fileExist(appPath) {
		if !options.force {
			r, err := command.PromptForConfirmation(ctx, dockerCli.In(), dockerCli.Out(), appExistWarn)
			if err != nil {
				return "", err
			}
			if !r {
				return "", errdefs.Cancelled(errors.New("app install has been canceled"))
			}
		}
		if err := removeApp(dockerCli, binPath, appPath, options); err != nil {
			return "", err
		}
	}

	// for the default destination
	// if there is only one file, create symlink for the file
	if fp, err := oneChild(dir); err == nil && fp != "" {
		appName := options.name
		if appName == "" {
			appName = makeAppName(fp)
		}
		if err := validateName(appName); err != nil {
			return "", err
		}

		link, err := installOne(appName, dir, fp, binPath, appPath)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(dockerCli.Out(), "App installed: %s\n", link)
		return link, nil
	}

	// if there is a run file, create symlink for the run file.
	if fp, err := locateFile(dir, runnerName); err == nil && fp != "" {
		appName := options.name
		if appName == "" {
			appName = makeAppName(appPath)
		}
		if err := validateName(appName); err != nil {
			return "", err
		}

		link, err := installRunFile(appName, dir, fp, binPath, appPath)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(dockerCli.Out(), "App installed: %s\n", link)
		return link, nil
	}

	// custom install
	if err := installCustom(dir, appPath, options); err != nil {
		return "", err
	}

	fmt.Fprintf(dockerCli.Out(), "App installer ran successfully\n")
	return "", nil
}

// removeApp removes the existing app
func removeApp(dockerCli cliAdapter, binPath, appPath string, options *AppOptions) error {
	envs, _ := options.makeEnvs()
	runUninstaller(dockerCli, appPath, envs)

	if err := os.RemoveAll(appPath); err != nil {
		return err
	}
	targets, err := findSymlinks(binPath)
	if err != nil {
		return err
	}
	cleanupSymlink(dockerCli, appPath, targets)
	return nil
}

// installOne creates a symlink to the only file in appPath
func installOne(appName, egress, runPath, binPath, appPath string) (string, error) {
	link := filepath.Join(binPath, appName)
	target := filepath.Join(appPath, appName)
	return install(link, target, egress, appPath)
}

// installRunFile creates a symlink to the run file in appPath
func installRunFile(appName, egress, runPath, binPath, appPath string) (string, error) {
	link := filepath.Join(binPath, appName)
	target := filepath.Join(appPath, filepath.Base(runPath))
	return install(link, target, egress, appPath)
}

// instal creates a symlink to the target file
func install(link, target, egress, appPath string) (string, error) {
	if ok, err := isSymlinkOK(link, target); err != nil {
		return "", err
	} else {
		if !ok {
			return "", fmt.Errorf("another app/version file exists: %s", link)
		}
		if err := os.Remove(link); err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(appPath), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(egress, appPath); err != nil {
		return "", err
	}

	// make target executable
	if err := os.Chmod(target, 0o755); err != nil {
		return "", err
	}

	if err := os.Symlink(target, link); err != nil {
		return "", err
	}
	return link, nil
}

func installCustom(dir string, appPath string, options *AppOptions) error {
	if err := os.MkdirAll(filepath.Dir(appPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(dir, appPath); err != nil {
		return err
	}

	// optionally run the installer if it exists
	installer := filepath.Join(appPath, installerName)
	if !fileExist(installer) {
		return nil
	}
	if err := os.Chmod(installer, 0o755); err != nil {
		return err
	}

	return launch(installer, options)
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// makeAppName derives the app name from the base name of the path
// after removing the version and extension
func makeAppName(path string) string {
	n := filepath.Base(path)
	n = strings.SplitN(n, "@", 2)[0]
	n = strings.SplitN(n, ".", 2)[0]
	return n
}
