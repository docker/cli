// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package system

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/inspect"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	format      string
	inspectType string
	size        bool
	ids         []string
}

// NewInspectCommand creates a new cobra.Command for `docker inspect`
func NewInspectCommand(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] NAME|ID [NAME|ID...]",
		Short: "Return low-level information on Docker objects",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ids = args
			return runInspect(cmd.Context(), dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	flags.StringVar(&opts.inspectType, "type", "", "Return JSON for specified type")
	flags.BoolVarP(&opts.size, "size", "s", false, "Display total file sizes if the type is container")

	return cmd
}

func runInspect(ctx context.Context, dockerCli command.Cli, opts inspectOptions) error {
	var elementSearcher inspect.GetRefFunc
	switch opts.inspectType {
	case "", "config", "container", "image", "network", "node", "plugin", "secret", "service", "task", "volume":
		elementSearcher = inspectAll(ctx, dockerCli, opts.size, opts.inspectType)
	default:
		return errors.Errorf("%q is not a valid value for --type", opts.inspectType)
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
		return dockerCli.Client().ServiceInspectWithRaw(ctx, ref, types.ServiceInspectOptions{InsertDefaults: true})
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

func inspectAll(ctx context.Context, dockerCLI command.Cli, getSize bool, typeConstraint string) inspect.GetRefFunc {
	inspectAutodetect := []struct {
		objectType      string
		isSizeSupported bool
		isSwarmObject   bool
		objectInspector func(string) (any, []byte, error)
	}{
		{
			objectType:      "container",
			isSizeSupported: true,
			objectInspector: inspectContainers(ctx, dockerCLI, getSize),
		},
		{
			objectType:      "image",
			objectInspector: inspectImages(ctx, dockerCLI),
		},
		{
			objectType:      "network",
			objectInspector: inspectNetwork(ctx, dockerCLI),
		},
		{
			objectType:      "volume",
			objectInspector: inspectVolume(ctx, dockerCLI),
		},
		{
			objectType:      "service",
			isSwarmObject:   true,
			objectInspector: inspectService(ctx, dockerCLI),
		},
		{
			objectType:      "task",
			isSwarmObject:   true,
			objectInspector: inspectTasks(ctx, dockerCLI),
		},
		{
			objectType:      "node",
			isSwarmObject:   true,
			objectInspector: inspectNode(ctx, dockerCLI),
		},
		{
			objectType:      "plugin",
			objectInspector: inspectPlugin(ctx, dockerCLI),
		},
		{
			objectType:      "secret",
			isSwarmObject:   true,
			objectInspector: inspectSecret(ctx, dockerCLI),
		},
		{
			objectType:      "config",
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
