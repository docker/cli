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
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/gdamore/tcell"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type statsOptions struct {
	all        bool
	noStream   bool
	noTrunc    bool
	tcell      bool
	format     string
	containers []string
}

// NewStatsCommand creates a new cobra.Command for `docker stats`
func NewStatsCommand(dockerCli command.Cli) *cobra.Command {
	var opts statsOptions

	cmd := &cobra.Command{
		Use:   "stats [OPTIONS] [CONTAINER...]",
		Short: "Display a live stream of container(s) resource usage statistics",
		Args:  cli.RequiresMinArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runStats(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.all, "all", "a", false, "Show all containers (default shows just running)")
	flags.BoolVar(&opts.noStream, "no-stream", false, "Disable streaming stats and only pull the first result")
	flags.BoolVar(&opts.noTrunc, "no-trunc", false, "Do not truncate output")
	flags.BoolVar(&opts.tcell, "tcell", false, "Use tcell to print. Use 'q' to stop displaying then 'Ctrl^C' to quit.")
	flags.StringVar(&opts.format, "format", "", "Pretty-print images using a Go template")
	return cmd
}

// runStats displays a live stream of resource usage statistics for one or more containers.
// This shows real-time information on CPU usage, memory usage, and network I/O.
// nolint: gocyclo
func runStats(dockerCli command.Cli, opts *statsOptions) error {
	showAll := len(opts.containers) == 0
	closeChan := make(chan error)

	ctx := context.Background()

	// monitorContainerEvents watches for container creation and removal (only
	// used when calling `docker stats` without arguments).
	monitorContainerEvents := func(started chan<- struct{}, c chan events.Message) {
		f := filters.NewArgs()
		f.Add("type", "container")
		options := types.EventsOptions{
			Filters: f,
		}

		eventq, errq := dockerCli.Client().Events(ctx, options)

		// Whether we successfully subscribed to eventq or not, we can now
		// unblock the main goroutine.
		close(started)

		for {
			select {
			case event := <-eventq:
				c <- event
			case err := <-errq:
				closeChan <- err
				return
			}
		}
	}

	// Get the daemonOSType if not set already
	if daemonOSType == "" {
		svctx := context.Background()
		sv, err := dockerCli.Client().ServerVersion(svctx)
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
		options := types.ContainerListOptions{
			All: opts.all,
		}
		cs, err := dockerCli.Client().ContainerList(ctx, options)
		if err != nil {
			closeChan <- err
		}
		for _, container := range cs {
			s := NewStats(container.ID[:12])
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, s, dockerCli.Client(), !opts.noStream, waitFirst)
			}
		}
	}

	if showAll {
		// If no names were specified, start a long running goroutine which
		// monitors container events. We make sure we're subscribed before
		// retrieving the list of running containers to avoid a race where we
		// would "miss" a creation.
		started := make(chan struct{})
		eh := command.InitEventHandler()
		eh.Handle("create", func(e events.Message) {
			if opts.all {
				s := NewStats(e.ID[:12])
				if cStats.add(s) {
					waitFirst.Add(1)
					go collect(ctx, s, dockerCli.Client(), !opts.noStream, waitFirst)
				}
			}
		})

		eh.Handle("start", func(e events.Message) {
			s := NewStats(e.ID[:12])
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, s, dockerCli.Client(), !opts.noStream, waitFirst)
			}
		})

		eh.Handle("die", func(e events.Message) {
			if !opts.all {
				cStats.remove(e.ID[:12])
			}
		})

		eventChan := make(chan events.Message)
		go eh.Watch(eventChan)
		go monitorContainerEvents(started, eventChan)
		defer close(eventChan)
		<-started

		// Start a short-lived goroutine to retrieve the initial list of
		// containers.
		getContainerList()
	} else {
		// Artificially send creation events for the containers we were asked to
		// monitor (same code path than we use when monitoring all containers).
		for _, name := range opts.containers {
			s := NewStats(name)
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, s, dockerCli.Client(), !opts.noStream, waitFirst)
			}
		}

		// We don't expect any asynchronous errors: closeChan can be closed.
		close(closeChan)

		// Do a quick pause to detect any error with the provided list of
		// container names.
		time.Sleep(1500 * time.Millisecond)
		var errs []string
		cStats.mu.Lock()
		for _, c := range cStats.cs {
			if err := c.GetError(); err != nil {
				errs = append(errs, err.Error())
			}
		}
		cStats.mu.Unlock()
		if len(errs) > 0 {
			return errors.New(strings.Join(errs, "\n"))
		}
	}

	// before print to screen, make sure each container get at least one valid stat data
	waitFirst.Wait()
	format := opts.format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().StatsFormat) > 0 {
			format = dockerCli.ConfigFile().StatsFormat
		} else {
			format = formatter.TableFormatKey
		}
	}
	statsCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewStatsFormat(format, daemonOSType),
	}

	if opts.tcell {
		tc := dockerCli.Tcell()

		if tc == nil {
			fmt.Fprintf(dockerCli.Out(), "tc is nil, there was a problem during its initialization.\n")

			return nil
		}

		tc.Init()
		statsCtx.Output = tc

		// Goroutine used to check event.
		go func(tc *streams.Tcell) {
			for {
				screen := tc.Screen()
				event := screen.PollEvent()

				/*
				 * This snippet was highly inspired by:
				 * https://github.com/gdamore/tcell/blob/master/_demos/unicode.go#L173
				 */
				switch event := event.(type) {
				case *tcell.EventKey:
					/*
					 * If user presses 'q' we finish the screen and terminate the
					 * goroutine.
					 */
					if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
						screen.Fini()

						return
					}
				case *tcell.EventResize:
					/*
					 * If a resize event is received (because user resized the windows)
					 * we need to update value of width.
					 */
					width, _ := event.Size()

					tc.Resize(width)
				}
			}
		}(tc)
	}
	cleanScreen := func() {
		if !opts.noStream && !opts.tcell {
			fmt.Fprint(dockerCli.Out(), "\033[2J")
			fmt.Fprint(dockerCli.Out(), "\033[H")
		}
	}

	var err error
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		/*
		 * If tcell option is not used we will clear screen by using Ctrl^L escape
		 * sequence.
		 * Otherwise we need to use specific Tcell function to effectively display
		 * things.
		 */
		if !opts.tcell {
			cleanScreen()
		} else {
			dockerCli.Tcell().Display()
		}
		ccstats := []StatsEntry{}
		cStats.mu.Lock()
		for _, c := range cStats.cs {
			ccstats = append(ccstats, c.GetStatistics())
		}
		cStats.mu.Unlock()
		if err = statsFormatWrite(statsCtx, ccstats, daemonOSType, !opts.noTrunc); err != nil {
			break
		}
		if len(cStats.cs) == 0 && !showAll {
			break
		}
		if opts.noStream {
			break
		}
		select {
		case err, ok := <-closeChan:
			if ok {
				if err != nil {
					// this is suppressing "unexpected EOF" in the cli when the
					// daemon restarts so it shutdowns cleanly
					if err == io.ErrUnexpectedEOF {
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
