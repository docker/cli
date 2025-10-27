package container

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// StatsOptions defines options for [RunStats].
type StatsOptions struct {
	// All allows including both running and stopped containers. The default
	// is to only include running containers.
	All bool

	// NoStream disables streaming stats. If enabled, stats are collected once,
	// and the result is printed.
	NoStream bool

	// NoTrunc disables truncating the output. The default is to truncate
	// output such as container-IDs.
	NoTrunc bool

	// Format is a custom template to use for presenting the stats.
	// Refer to [flagsHelper.FormatHelp] for accepted formats.
	Format string

	// Containers is the list of container names or IDs to include in the stats.
	// If empty, all containers are included. It is mutually exclusive with the
	// Filters option, and an error is produced if both are set.
	Containers []string

	// Filters provides optional filters to filter the list of containers and their
	// associated container-events to include in the stats if no list of containers
	// is set. If no filter is provided, all containers are included. Filters and
	// Containers are currently mutually exclusive, and setting both options
	// produces an error.
	//
	// These filters are used both to collect the initial list of containers and
	// to refresh the list of containers based on container-events, accepted
	// filters are limited to the intersection of filters accepted by "events"
	// and "container list".
	//
	// Currently only "label" / "label=value" filters are accepted. Additional
	// filter options may be added in future (within the constraints described
	// above), but may require daemon-side validation as the list of accepted
	// filters can differ between daemon- and API versions.
	Filters client.Filters
}

// newStatsCommand creates a new [cobra.Command] for "docker container stats".
func newStatsCommand(dockerCLI command.Cli) *cobra.Command {
	options := StatsOptions{}

	cmd := &cobra.Command{
		Use:   "stats [OPTIONS] [CONTAINER...]",
		Short: "Display a live stream of container(s) resource usage statistics",
		Args:  cli.RequiresMinArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Containers = args
			return RunStats(cmd.Context(), dockerCLI, &options)
		},
		Annotations: map[string]string{
			"aliases": "docker container stats, docker stats",
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, false),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.All, "all", "a", false, "Show all containers (default shows just running)")
	flags.BoolVar(&options.NoStream, "no-stream", false, "Disable streaming stats and only pull the first result")
	flags.BoolVar(&options.NoTrunc, "no-trunc", false, "Do not truncate output")
	flags.StringVar(&options.Format, "format", "", flagsHelper.FormatHelp)
	return cmd
}

// acceptedStatsFilters is the list of filters accepted by [RunStats] (through
// the [StatsOptions.Filters] option).
//
// TODO(thaJeztah): don't hard-code the list of accept filters, and expand
// to the intersection of filters accepted by both "container list" and
// "system events". Validating filters may require an initial API call
// to both endpoints ("container list" and "system events").
var acceptedStatsFilters = map[string]bool{
	"label": true,
}

// RunStats displays a live stream of resource usage statistics for one or more containers.
// This shows real-time information on CPU usage, memory usage, and network I/O.
//
//nolint:gocyclo
func RunStats(ctx context.Context, dockerCLI command.Cli, options *StatsOptions) error {
	apiClient := dockerCLI.Client()

	// Get the daemonOSType to handle platform-specific stats fields.
	// This value is used as a fallback for docker < v29, which did not
	// include the OSType field per stats.
	daemonOSType = dockerCLI.ServerInfo().OSType

	// waitFirst is a WaitGroup to wait first stat data's reach for each container
	waitFirst := &sync.WaitGroup{}
	// closeChan is used to collect errors from goroutines. It uses a small buffer
	// to avoid blocking sends when sends occur after closeChan is set to nil or
	// after the reader has exited, preventing deadlocks.
	closeChan := make(chan error, 4)
	cStats := stats{}

	showAll := len(options.Containers) == 0
	if showAll {
		// If no names were specified, start a long-running goroutine which
		// monitors container events. We make sure we're subscribed before
		// retrieving the list of running containers to avoid a race where we
		// would "miss" a creation.
		started := make(chan struct{})

		if options.Filters == nil {
			options.Filters = make(client.Filters)
		}

		// FIXME(thaJeztah): any way we can (and should?) validate allowed filters?
		for filter := range options.Filters {
			if _, ok := acceptedStatsFilters[filter]; !ok {
				return errdefs.ErrInvalidArgument.WithMessage("invalid filter '" + filter + "'")
			}
		}

		eh := newEventHandler()
		if options.All {
			eh.setHandler(events.ActionCreate, func(e events.Message) {
				if s := NewStats(e.Actor.ID); cStats.add(s) {
					waitFirst.Add(1)
					log.G(ctx).WithFields(map[string]any{
						"event":     e.Action,
						"container": e.Actor.ID,
					}).Debug("collecting stats for container")
					go collect(ctx, s, apiClient, !options.NoStream, waitFirst)
				}
			})
		}

		eh.setHandler(events.ActionStart, func(e events.Message) {
			if s := NewStats(e.Actor.ID); cStats.add(s) {
				waitFirst.Add(1)
				log.G(ctx).WithFields(map[string]any{
					"event":     e.Action,
					"container": e.Actor.ID,
				}).Debug("collecting stats for container")
				go collect(ctx, s, apiClient, !options.NoStream, waitFirst)
			}
		})

		if !options.All {
			eh.setHandler(events.ActionDie, func(e events.Message) {
				log.G(ctx).WithFields(map[string]any{
					"event":     e.Action,
					"container": e.Actor.ID,
				}).Debug("stop collecting stats for container")
				cStats.remove(e.Actor.ID)
			})
		}

		// monitorContainerEvents watches for container creation and removal (only
		// used when calling `docker stats` without arguments).
		monitorContainerEvents := func(started chan<- struct{}, c chan events.Message, stopped <-chan struct{}) {
			// Create a copy of the custom filters so that we don't mutate
			// the original set of filters. Custom filters are used both
			// to list containers and to filter events, but the "type" filter
			// is not valid for filtering containers.
			f := options.Filters.Clone().Add("type", string(events.ContainerEventType))
			eventChan, errChan := apiClient.Events(ctx, client.EventsListOptions{
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
				case <-ctx.Done():
					return
				case event := <-eventChan:
					c <- event
				case err := <-errChan:
					// Prevent blocking if closeChan is full or unread
					select {
					case closeChan <- err:
					default:
						// drop if not read; avoids deadlock
					}
					return
				}
			}
		}

		eventChan := make(chan events.Message)
		go eh.watch(eventChan)
		stopped := make(chan struct{})
		go monitorContainerEvents(started, eventChan, stopped)
		defer close(stopped)
		<-started

		// Fetch the initial list of containers and collect stats for them.
		// After the initial list was collected, we start listening for events
		// to refresh the list of containers.
		cs, err := apiClient.ContainerList(ctx, client.ContainerListOptions{
			All:     options.All,
			Filters: options.Filters,
		})
		if err != nil {
			return err
		}
		for _, ctr := range cs {
			if s := NewStats(ctr.ID); cStats.add(s) {
				waitFirst.Add(1)
				log.G(ctx).WithFields(map[string]any{
					"container": ctr.ID,
				}).Debug("collecting stats for container")
				go collect(ctx, s, apiClient, !options.NoStream, waitFirst)
			}
		}

		// make sure each container get at least one valid stat data
		waitFirst.Wait()
	} else {
		// TODO(thaJeztah): re-implement options.Containers as a filter so that
		// only a single code-path is needed, and custom filters can be combined
		// with a list of container names/IDs.

		if len(options.Filters) > 0 {
			return errors.New("filtering is not supported when specifying a list of containers")
		}

		// Create the list of containers, and start collecting stats for all
		// containers passed.
		for _, ctr := range options.Containers {
			if s := NewStats(ctr); cStats.add(s) {
				waitFirst.Add(1)
				log.G(ctx).WithFields(map[string]any{
					"container": ctr,
				}).Debug("collecting stats for container")
				go collect(ctx, s, apiClient, !options.NoStream, waitFirst)
			}
		}

		// We don't expect any asynchronous errors: closeChan can be closed and disabled.
		close(closeChan)
		closeChan = nil

		// make sure each container get at least one valid stat data
		waitFirst.Wait()

		var errs []error
		cStats.mu.RLock()
		for _, c := range cStats.cs {
			if err := c.GetError(); err != nil {
				errs = append(errs, err)
			}
		}
		cStats.mu.RUnlock()
		if err := errors.Join(errs...); err != nil {
			return err
		}
	}

	format := options.Format
	if format == "" {
		if len(dockerCLI.ConfigFile().StatsFormat) > 0 {
			format = dockerCLI.ConfigFile().StatsFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	// Buffer to store formatted stats text.
	// Once formatted, it will be printed in one write to avoid screen flickering.
	var statsTextBuffer bytes.Buffer

	statsCtx := formatter.Context{
		Output: &statsTextBuffer,
		Format: NewStatsFormat(format, daemonOSType),
	}

	if options.NoStream {
		cStats.mu.RLock()
		ccStats := make([]StatsEntry, 0, len(cStats.cs))
		for _, c := range cStats.cs {
			ccStats = append(ccStats, c.GetStatistics())
		}
		cStats.mu.RUnlock()

		if len(ccStats) == 0 {
			return nil
		}
		if err := statsFormatWrite(statsCtx, ccStats, daemonOSType, !options.NoTrunc); err != nil {
			return err
		}
		_, _ = fmt.Fprint(dockerCLI.Out(), statsTextBuffer.String())
		return nil
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cStats.mu.RLock()
			ccStats := make([]StatsEntry, 0, len(cStats.cs))
			for _, c := range cStats.cs {
				ccStats = append(ccStats, c.GetStatistics())
			}
			cStats.mu.RUnlock()

			// Start by moving the cursor to the top-left
			_, _ = fmt.Fprint(&statsTextBuffer, "\033[H")

			if err := statsFormatWrite(statsCtx, ccStats, daemonOSType, !options.NoTrunc); err != nil {
				return err
			}

			for _, line := range strings.Split(statsTextBuffer.String(), "\n") {
				// In case the new text is shorter than the one we are writing over,
				// we'll append the "erase line" escape sequence to clear the remaining text.
				_, _ = fmt.Fprintln(&statsTextBuffer, line, "\033[K")
			}
			// We might have fewer containers than before, so let's clear the remaining text
			_, _ = fmt.Fprint(&statsTextBuffer, "\033[J")

			_, _ = fmt.Fprint(dockerCLI.Out(), statsTextBuffer.String())
			statsTextBuffer.Reset()

			if len(ccStats) == 0 && !showAll {
				return nil
			}
		case err, ok := <-closeChan:
			if !ok || err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				// Suppress "unexpected EOF" errors in the CLI so that
				// it shuts down cleanly when the daemon restarts.
				return nil
			}
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// newEventHandler initializes and returns an eventHandler
func newEventHandler() *eventHandler {
	return &eventHandler{handlers: make(map[events.Action]func(events.Message))}
}

// eventHandler allows for registering specific events to setHandler.
type eventHandler struct {
	handlers map[events.Action]func(events.Message)
}

func (eh *eventHandler) setHandler(action events.Action, handler func(events.Message)) {
	eh.handlers[action] = handler
}

// watch ranges over the passed in event chan and processes the events based on the
// handlers created for a given action.
// To stop watching, close the event chan.
func (eh *eventHandler) watch(c <-chan events.Message) {
	for e := range c {
		h, exists := eh.handlers[e.Action]
		if !exists {
			continue
		}
		if e.Actor.ID == "" {
			log.G(context.TODO()).WithField("event", e).Errorf("event handler: received %s event with empty ID", e.Action)
			continue
		}

		log.G(context.TODO()).WithField("event", e).Debugf("event handler: received %s event for: %s", e.Action, e.Actor.ID)
		go h(e)
	}
}
