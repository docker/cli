// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

package system

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/inspect"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type objectType = string

const (
	typeConfig    objectType = "config"
	typeContainer objectType = "container"
	typeImage     objectType = "image"
	typeNetwork   objectType = "network"
	typeNode      objectType = "node"
	typePlugin    objectType = "plugin"
	typeSecret    objectType = "secret"
	typeService   objectType = "service"
	typeTask      objectType = "task"
	typeVolume    objectType = "volume"
)

var allTypes = []objectType{
	typeConfig,
	typeContainer,
	typeImage,
	typeNetwork,
	typeNode,
	typePlugin,
	typeSecret,
	typeService,
	typeTask,
	typeVolume,
}

type inspectOptions struct {
	format     string
	objectType objectType
	size       bool
	ids        []string
}

// newInspectCommand creates a new cobra.Command for `docker inspect`
func newInspectCommand(dockerCLI command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] NAME|ID [NAME|ID...]",
		Short: "Return low-level information on Docker objects",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ids = args
			if cmd.Flags().Changed("type") && opts.objectType == "" {
				return fmt.Errorf(`type is empty: must be one of "%s"`, strings.Join(allTypes, `", "`))
			}
			return runInspect(cmd.Context(), dockerCLI, opts)
		},
		// TODO(thaJeztah): should we consider adding completion for common object-types? (images, containers?)
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	flags.StringVar(&opts.objectType, "type", "", "Only inspect objects of the given type")
	flags.BoolVarP(&opts.size, "size", "s", false, "Display total file sizes if the type is container")

	_ = cmd.RegisterFlagCompletionFunc("type", completion.FromList(allTypes...))
	flags.VisitAll(func(flag *pflag.Flag) {
		// Set a default completion function if none was set. We don't look
		// up if it does already have one set, because Cobra does this for
		// us, and returns an error (which we ignore for this reason).
		_ = cmd.RegisterFlagCompletionFunc(flag.Name, completion.NoComplete)
	})
	return cmd
}

func runInspect(ctx context.Context, dockerCli command.Cli, opts inspectOptions) error {
	var elementSearcher inspect.GetRefFunc
	switch opts.objectType {
	case "", typeConfig, typeContainer, typeImage, typeNetwork, typeNode,
		typePlugin, typeSecret, typeService, typeTask, typeVolume:
		elementSearcher = inspectAll(ctx, dockerCli, opts.size, opts.objectType)
	default:
		return errors.Errorf(`unknown type: %q: must be one of "%s"`, opts.objectType, strings.Join(allTypes, `", "`))
	}
	return inspect.Inspect(dockerCli.Out(), opts.ids, opts.format, elementSearcher)
}

func inspectContainers(ctx context.Context, dockerCli command.Cli, getSize bool) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		return dockerCli.Client().ContainerInspectWithRaw(ctx, ref, getSize)
	}
}

func inspectImages(ctx context.Context, dockerCli command.Cli) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		var buf bytes.Buffer
		resp, err := dockerCli.Client().ImageInspect(ctx, ref, client.ImageInspectWithRawResponse(&buf))
		if err != nil {
			return image.InspectResponse{}, nil, err
		}
		return resp, buf.Bytes(), err
	}
}

func inspectNetwork(ctx context.Context, dockerCli command.Cli) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		return dockerCli.Client().NetworkInspectWithRaw(ctx, ref, network.InspectOptions{})
	}
}

func inspectNode(ctx context.Context, dockerCli command.Cli) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		return dockerCli.Client().NodeInspectWithRaw(ctx, ref)
	}
}

func inspectService(ctx context.Context, dockerCli command.Cli) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		// Service inspect shows defaults values in empty fields.
		return dockerCli.Client().ServiceInspectWithRaw(ctx, ref, swarm.ServiceInspectOptions{InsertDefaults: true})
	}
}

func inspectTasks(ctx context.Context, dockerCli command.Cli) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		return dockerCli.Client().TaskInspectWithRaw(ctx, ref)
	}
}

func inspectVolume(ctx context.Context, dockerCli command.Cli) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		return dockerCli.Client().VolumeInspectWithRaw(ctx, ref)
	}
}

func inspectPlugin(ctx context.Context, dockerCli command.Cli) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		return dockerCli.Client().PluginInspectWithRaw(ctx, ref)
	}
}

func inspectSecret(ctx context.Context, dockerCli command.Cli) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		return dockerCli.Client().SecretInspectWithRaw(ctx, ref)
	}
}

func inspectConfig(ctx context.Context, dockerCLI command.Cli) inspect.GetRefFunc {
	return func(ref string) (any, []byte, error) {
		return dockerCLI.Client().ConfigInspectWithRaw(ctx, ref)
	}
}

func inspectAll(ctx context.Context, dockerCLI command.Cli, getSize bool, typeConstraint objectType) inspect.GetRefFunc {
	inspectAutodetect := []struct {
		objectType      objectType
		isSizeSupported bool
		isSwarmObject   bool
		objectInspector func(string) (any, []byte, error)
	}{
		{
			objectType:      typeContainer,
			isSizeSupported: true,
			objectInspector: inspectContainers(ctx, dockerCLI, getSize),
		},
		{
			objectType:      typeImage,
			objectInspector: inspectImages(ctx, dockerCLI),
		},
		{
			objectType:      typeNetwork,
			objectInspector: inspectNetwork(ctx, dockerCLI),
		},
		{
			objectType:      typeVolume,
			objectInspector: inspectVolume(ctx, dockerCLI),
		},
		{
			objectType:      typeService,
			isSwarmObject:   true,
			objectInspector: inspectService(ctx, dockerCLI),
		},
		{
			objectType:      typeTask,
			isSwarmObject:   true,
			objectInspector: inspectTasks(ctx, dockerCLI),
		},
		{
			objectType:      typeNode,
			isSwarmObject:   true,
			objectInspector: inspectNode(ctx, dockerCLI),
		},
		{
			objectType:      typePlugin,
			objectInspector: inspectPlugin(ctx, dockerCLI),
		},
		{
			objectType:      typeSecret,
			isSwarmObject:   true,
			objectInspector: inspectSecret(ctx, dockerCLI),
		},
		{
			objectType:      typeConfig,
			isSwarmObject:   true,
			objectInspector: inspectConfig(ctx, dockerCLI),
		},
	}

	// isSwarmManager does an Info API call to verify that the daemon is
	// a swarm manager.
	isSwarmManager := func() bool {
		info, err := dockerCLI.Client().Info(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(dockerCLI.Err(), err)
			return false
		}
		return info.Swarm.ControlAvailable
	}

	return func(ref string) (any, []byte, error) {
		const (
			swarmSupportUnknown = iota
			swarmSupported
			swarmUnsupported
		)

		isSwarmSupported := swarmSupportUnknown

		for _, inspectData := range inspectAutodetect {
			if typeConstraint != "" && inspectData.objectType != typeConstraint {
				continue
			}
			if typeConstraint == "" && inspectData.isSwarmObject {
				if isSwarmSupported == swarmSupportUnknown {
					if isSwarmManager() {
						isSwarmSupported = swarmSupported
					} else {
						isSwarmSupported = swarmUnsupported
					}
				}
				if isSwarmSupported == swarmUnsupported {
					continue
				}
			}
			v, raw, err := inspectData.objectInspector(ref)
			if err != nil {
				if typeConstraint == "" && isErrSkippable(err) {
					continue
				}
				return v, raw, err
			}
			if getSize && !inspectData.isSizeSupported {
				_, _ = fmt.Fprintln(dockerCLI.Err(), "WARNING: --size ignored for", inspectData.objectType)
			}
			return v, raw, err
		}
		return nil, nil, errors.Errorf("Error: No such object: %s", ref)
	}
}

func isErrSkippable(err error) bool {
	return errdefs.IsNotFound(err) ||
		strings.Contains(err.Error(), "not supported") ||
		strings.Contains(err.Error(), "invalid reference format")
}
