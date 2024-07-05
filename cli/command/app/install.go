package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/image"
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

	// install specific build args
	addBuildArgs(flags)

	return options
}

func addBuildArgs(flags *pflag.FlagSet) {
	flag := flags.Lookup("build-arg")
	if flag != nil {
		flag.Value.Set("HOSTOS=" + runtime.GOOS)
		flag.Value.Set("HOSTARCH=" + runtime.GOARCH)
	}
}

func installApp(ctx context.Context, adapter cliAdapter, flags *pflag.FlagSet, options *AppOptions) error {
	if err := validateAppOptions(options); err != nil {
		return err
	}

	dir, err := runInstall(ctx, adapter, flags, options)
	if err != nil {
		return err
	}

	bin, err := runPostInstall(adapter, dir, options)
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

// runInstall calls the build, run, and cp commands
func runInstall(ctx context.Context, dockerCli cliAdapter, flags *pflag.FlagSet, options *AppOptions) (string, error) {
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

func runPostInstall(dockerCli cliAdapter, dir string, options *AppOptions) (string, error) {
	if !options.isDockerAppBase() {
		return "", installCustom(dir, options.destination, options)
	}

	// for the default destination

	// if there is only one file, create symlink for the file
	if fp, err := oneChild(dir); err == nil && fp != "" {
		binPath := options.binPath()
		appPath, err := options.appPath()
		if err != nil {
			return "", err
		}

		link, err := installOne(dir, fp, binPath, appPath)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(dockerCli.Out(), "App installed: %s\n", link)
		return link, nil
	}

	// if there is a run file, create symlink for the run file.
	if fp, err := locateFile(dir, runnerName); err == nil && fp != "" {
		binPath := options.binPath()
		appPath, err := options.appPath()
		if err != nil {
			return "", err
		}

		link, err := installRunFile(dir, fp, binPath, appPath)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(dockerCli.Out(), "App installed: %s\n", link)
		return link, nil
	}

	// custom install
	appPath, err := options.appPath()
	if err != nil {
		return "", err
	}
	if err := installCustom(dir, appPath, options); err != nil {
		return "", err
	}

	fmt.Fprintf(dockerCli.Out(), "App installer ran successfully\n")
	return "", nil
}

// installOne creates a symlink to the only file in appPath
func installOne(egress, runPath, binPath, appPath string) (string, error) {
	appName := filepath.Base(runPath)
	if err := validateName(appName); err != nil {
		return "", err
	}
	link := filepath.Join(binPath, appName)
	target := filepath.Join(appPath, appName)
	return install(link, target, egress, binPath, appPath)
}

// installRunFile creates a symlink to the run file in appPath
// use the base name as the app name
func installRunFile(egress, runPath, binPath, appPath string) (string, error) {
	appName := filepath.Base(appPath)
	if err := validateName(appName); err != nil {
		return "", err
	}
	link := filepath.Join(binPath, appName)
	target := filepath.Join(appPath, filepath.Base(runPath))
	return install(link, target, egress, binPath, appPath)
}

// instal creates a symlink to the target file
func install(link, target, egress, binPath, appPath string) (string, error) {
	if _, err := os.Stat(appPath); err == nil {
		return "", fmt.Errorf("app package exists: %s! Try again after removing it", appPath)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("installation failed: %w", err)
	}

	if ok, err := isSymlinkToOK(link, target); err != nil {
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

	if err := os.MkdirAll(binPath, 0o755); err != nil {
		return "", err
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
	exist := func(p string) bool {
		_, err := os.Stat(p)
		if os.IsNotExist(err) {
			return false
		}
		return err == nil
	}

	if exist(appPath) {
		return fmt.Errorf("destination exists: %s", appPath)
	}

	if err := os.MkdirAll(filepath.Dir(appPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(dir, appPath); err != nil {
		return err
	}

	// optionally run the installer if it exists
	installer := filepath.Join(appPath, installerName)
	if !exist(installer) {
		return nil
	}
	if err := os.Chmod(installer, 0o755); err != nil {
		return err
	}

	return launch(installer, options)
}
