package pruner

import (
	"context"
	"fmt"
	"io"

	"github.com/moby/moby/api/types/prune"

	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/versions"
)

// ProgressPrinter prints confirmed deletions as they arrive from the daemon.
type ProgressPrinter struct {
	Stream  bool
	Print   func(prune.Object) error
	Started bool
	out     io.Writer
}

// NewProgressPrinter uses streaming when supported by the negotiated API version.
// Older daemons continue to use the final prune report.
func NewProgressPrinter(cli command.Cli, contentType ContentType) *ProgressPrinter {
	p := &ProgressPrinter{out: cli.Out()}
	if versions.LessThan(cli.CurrentVersion(), "1.56") {
		return p
	}
	headings := map[ContentType]string{
		TypeContainer:  "Deleted Containers:",
		TypeNetwork:    "Deleted Networks:",
		TypeVolume:     "Deleted Volumes:",
		TypeImage:      "Deleted Images:",
		TypeBuildCache: "Deleted build cache objects:",
	}
	p.Stream = true
	p.Print = func(progress prune.Object) error {
		if !p.Started {
			if _, err := fmt.Fprintln(p.out, headings[contentType]); err != nil {
				return err
			}
			p.Started = true
		}
		if contentType == TypeImage {
			_, err := fmt.Fprintf(p.out, "%s: %s\n", progress.Action, progress.ID)
			return err
		}
		_, err := fmt.Fprintln(p.out, progress.ID)
		return err
	}
	return p
}

// Finish preserves the blank line after a nonempty prune report.
func (p *ProgressPrinter) Finish() error {
	if !p.Started {
		return nil
	}
	_, err := fmt.Fprintln(p.out)
	return err
}

// Consume prints typed object events and returns the complete report.
func Consume[T any](ctx context.Context, response client.PruneResponse[T], printer *ProgressPrinter) (T, error) {
	defer response.Close()
	for object, err := range response.Events(ctx) {
		if err != nil {
			var zero T
			return zero, err
		}
		if err := printer.Print(object); err != nil {
			var zero T
			return zero, err
		}
	}
	return response.Wait(ctx)
}
