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

type statsOptions struct {
	all        bool
	noStream   bool
	noTrunc    bool
	format     string
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
	showAll := len(options.containers) == 0
	closeChan := make(chan error)
	apiClient := dockerCLI.Client()

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

	cStats := stats{}
	// getContainerList simulates creation event for all previously existing
	// containers (only used when calling `docker stats` without arguments).
	getContainerList := func() {
		cs, err := apiClient.ContainerList(ctx, container.ListOptions{
			All: options.all,
		})
		if err != nil {
			closeChan <- err
		}
		for _, ctr := range cs {
			s := NewStats(ctr.ID[:12])
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, s, apiClient, !options.noStream, waitFirst)
			}
		}
	}

	if showAll {
		// If no names were specified, start a long-running goroutine which
		// monitors container events. We make sure we're subscribed before
		// retrieving the list of running containers to avoid a race where we
		// would "miss" a creation.
		started := make(chan struct{})
		eh := command.InitEventHandler()
		eh.Handle(events.ActionCreate, func(e events.Message) {
			if options.all {
				s := NewStats(e.Actor.ID[:12])
				if cStats.add(s) {
					waitFirst.Add(1)
					go collect(ctx, s, apiClient, !options.noStream, waitFirst)
				}
			}
		})

		eh.Handle(events.ActionStart, func(e events.Message) {
			s := NewStats(e.Actor.ID[:12])
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, s, apiClient, !options.noStream, waitFirst)
			}
		})

		eh.Handle(events.ActionDie, func(e events.Message) {
			if !options.all {
				cStats.remove(e.Actor.ID[:12])
			}
		})

		eventChan := make(chan events.Message)
		go eh.Watch(eventChan)
		stopped := make(chan struct{})
		go monitorContainerEvents(started, eventChan, stopped)
		defer close(stopped)
		<-started

		// Start a short-lived goroutine to retrieve the initial list of
		// containers.
		getContainerList()

		// make sure each container get at least one valid stat data
		waitFirst.Wait()
	} else {
		// Artificially send creation events for the containers we were asked to
		// monitor (same code path than we use when monitoring all containers).
		for _, name := range options.containers {
			s := NewStats(name)
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
