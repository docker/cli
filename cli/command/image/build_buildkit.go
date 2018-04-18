package image

import (
	"io"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/urlutil"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/filesync"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/pkg/errors"
	"github.com/tonistiigi/fsutil"
	"golang.org/x/sync/errgroup"
)

func runBuildBuildKit(dockerCli command.Cli, options buildOptions) error {
	ctx := appcontext.Context()

	s, err := trySession(dockerCli, options.context)
	if err != nil {
		return err
	}
	if s == nil {
		return errors.Errorf("buildkit not supported by daemon")
	}

	remote := clientSessionRemote
	local := false
	switch {
	case options.contextFromStdin():
		return errors.Errorf("stdin not implemented")
	case isLocalDir(options.context):
		local = true
	case urlutil.IsGitURL(options.context):
		remote = options.context
	case urlutil.IsURL(options.context):
		remote = options.context
	default:
		return errors.Errorf("unable to prepare context: path %q not found", options.context)
	}

	// statusContext, cancelStatus := context.WithCancel(ctx)
	// defer cancelStatus()

	// if span := opentracing.SpanFromContext(ctx); span != nil {
	// 	statusContext = opentracing.ContextWithSpan(statusContext, span)
	// }

	if local {
		s.Allow(filesync.NewFSSyncProvider([]filesync.SyncedDir{
			{
				Name: "context",
				Dir:  options.context,
				Map:  resetUIDAndGID,
			},
			{
				Name: "dockerfile",
				Dir:  filepath.Dir(options.dockerfileName),
			},
		}))
	}

	s.Allow(authprovider.NewDockerAuthProvider())

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return s.Run(ctx, dockerCli.Client().DialSession)
	})

	eg.Go(func() error {
		defer func() { // make sure the Status ends cleanly on build errors
			s.Close()
		}()

		configFile := dockerCli.ConfigFile()
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
			Dockerfile:     filepath.Base(options.dockerfileName),
			ShmSize:        options.shmSize.Value(),
			Ulimits:        options.ulimits.GetList(),
			BuildArgs:      configFile.ParseProxyConfig(dockerCli.Client().DaemonHost(), options.buildArgs.GetAll()),
			// AuthConfigs:    authConfigs,
			Labels:        opts.ConvertKVStringsToMap(options.labels.GetAll()),
			CacheFrom:     options.cacheFrom,
			SecurityOpt:   options.securityOpt,
			NetworkMode:   options.networkMode,
			Squash:        options.squash,
			ExtraHosts:    options.extraHosts.GetAll(),
			Target:        options.target,
			RemoteContext: remote,
			Platform:      options.platform,
			SessionID:     "buildkit:" + s.ID(),
		}

		response, err := dockerCli.Client().ImageBuild(ctx, nil, buildOptions)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		if _, err := io.Copy(os.Stdout, response.Body); err != nil {
			return err
		}

		return nil
	})

	return eg.Wait()
}

func resetUIDAndGID(s *fsutil.Stat) bool {
	s.Uid = uint32(0)
	s.Gid = uint32(0)
	return true
}
