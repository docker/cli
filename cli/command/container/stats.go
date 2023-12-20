package container

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// statsOptions defines options for runStats.
type statsOptions struct {
	// all allows including both running and stopped containers. The default
	// is to only include running containers.
	all bool

	// noStream disables streaming stats. If enabled, stats are collected once,
	// and the result is printed.
	noStream bool

	// noTrunc disables truncating the output. The default is to truncate
	// output such as container-IDs.
	noTrunc bool

	// format is a custom template to use for presenting the stats.
	// Refer to [flagsHelper.FormatHelp] for accepted formats.
	format string

	// containers is the list of container names or IDs to include in the stats.
	// If empty, all containers are included.
	containers []string
}

// NewStatsCommand creates a new [cobra.Command] for "docker stats".
func NewStatsCommand(dockerCLI command.Cli) *cobra.Command {
	var options statsOptions

	cmd := &cobra.Command{
		Use:   "stats [OPTIONS] [CONTAINER...]",
		Short: "Display a live stream of container(s) resource usage statistics",
		Args:  cli.RequiresMinArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.containers = args
			return runStats(cmd.Context(), dockerCLI, &options)
		},
		Annotations: map[string]string{
			"aliases": "docker container stats, docker stats",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCLI, false),
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.all, "all", "a", false, "Show all containers (default shows just running)")
	flags.BoolVar(&options.noStream, "no-stream", false, "Disable streaming stats and only pull the first result")
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Do not truncate output")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	return cmd
}

// runStats displays a live stream of resource usage statistics for one or more containers.
// This shows real-time information on CPU usage, memory usage, and network I/O.
//
//nolint:gocyclo
func runStats(ctx context.Context, dockerCLI command.Cli, options *statsOptions) error {
	apiClient := dockerCLI.Client()

	// Get the daemonOSType if not set already
	if daemonOSType == "" {
		sv, err := apiClient.ServerVersion(ctx)
		if err != nil {
			return err
		}
		daemonOSType = sv.Os
	}

	// waitFirst is a WaitGroup to wait first stat data's reach for each container
	waitFirst := &sync.WaitGroup{}
	closeChan := make(chan error)
	cStats := stats{}

	showAll := len(options.containers) == 0
	if showAll {
		// If no names were specified, start a long-running goroutine which
		// monitors container events. We make sure we're subscribed before
		// retrieving the list of running containers to avoid a race where we
		// would "miss" a creation.
		started := make(chan struct{})
		eh := command.InitEventHandler()
		if options.all {
			eh.Handle(events.ActionCreate, func(e events.Message) {
				s := NewStats(e.Actor.ID[:12])
				if cStats.add(s) {
					waitFirst.Add(1)
					go collect(ctx, s, apiClient, !options.noStream, waitFirst)
				}
			})
		}

		eh.Handle(events.ActionStart, func(e events.Message) {
			s := NewStats(e.Actor.ID[:12])
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, s, apiClient, !options.noStream, waitFirst)
			}
		})

		if !options.all {
			eh.Handle(events.ActionDie, func(e events.Message) {
				cStats.remove(e.Actor.ID[:12])
			})
		}

		// monitorContainerEvents watches for container creation and removal (only
		// used when calling `docker stats` without arguments).
		monitorContainerEvents := func(started chan<- struct{}, c chan events.Message, stopped <-chan struct{}) {
			f := filters.NewArgs()
			f.Add("type", string(events.ContainerEventType))
			eventChan, errChan := apiClient.Events(ctx, types.EventsOptions{
				Filters: f,
			})

			// Whether we successfully subscribed to eventChan or not, we can now
			// unblock the main goroutine.
			close(started)
			defer close(c)

			for {
				select {
				case <-stopped:
					return
				case event := <-eventChan:
					c <- event
				case err := <-errChan:
					closeChan <- err
					return
				}
			}
		}

		eventChan := make(chan events.Message)
		go eh.Watch(eventChan)
		stopped := make(chan struct{})
		go monitorContainerEvents(started, eventChan, stopped)
		defer close(stopped)
		<-started

		// Fetch the initial list of containers and collect stats for them.
		// After the initial list was collected, we start listening for events
		// to refresh the list of containers.
		cs, err := apiClient.ContainerList(ctx, container.ListOptions{
			All: options.all,
		})
		if err != nil {
			return err
		}
		for _, ctr := range cs {
			s := NewStats(ctr.ID[:12])
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, s, apiClient, !options.noStream, waitFirst)
			}
		}

		// make sure each container get at least one valid stat data
		waitFirst.Wait()
	} else {
		// Create the list of containers, and start collecting stats for all
		// containers passed.
		for _, ctr := range options.containers {
			s := NewStats(ctr)
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, s, apiClient, !options.noStream, waitFirst)
			}
		}

		// We don't expect any asynchronous errors: closeChan can be closed.
		close(closeChan)

		// make sure each container get at least one valid stat data
		waitFirst.Wait()

		var errs []string
		cStats.mu.RLock()
		for _, c := range cStats.cs {
			if err := c.GetError(); err != nil {
				errs = append(errs, err.Error())
			}
		}
		cStats.mu.RUnlock()
		if len(errs) > 0 {
			return errors.New(strings.Join(errs, "\n"))
		}
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCLI.ConfigFile().StatsFormat) > 0 {
			format = dockerCLI.ConfigFile().StatsFormat
		} else {
			format = formatter.TableFormatKey
		}
	}
	statsCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: NewStatsFormat(format, daemonOSType),
	}
	cleanScreen := func() {
		if !options.noStream {
			_, _ = fmt.Fprint(dockerCLI.Out(), "\033[2J")
			_, _ = fmt.Fprint(dockerCLI.Out(), "\033[H")
		}
	}

	var err error
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		cleanScreen()
		var ccStats []StatsEntry
		cStats.mu.RLock()
		for _, c := range cStats.cs {
			ccStats = append(ccStats, c.GetStatistics())
		}
		cStats.mu.RUnlock()
		if err = statsFormatWrite(statsCtx, ccStats, daemonOSType, !options.noTrunc); err != nil {
			break
		}
		if len(cStats.cs) == 0 && !showAll {
			break
		}
		if options.noStream {
			break
		}
		select {
		case err, ok := <-closeChan:
			if ok {
				if err != nil {
					// Suppress "unexpected EOF" errors in the CLI so that
					// it shuts down cleanly when the daemon restarts.
					if errors.Is(err, io.ErrUnexpectedEOF) {
						return nil
					}
					return err
				}
			}
		default:
			// just skip
		}
	}
	return err
}
