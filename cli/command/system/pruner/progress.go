package pruner

import (
	"fmt"
	"io"

	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/versions"
)

// ProgressPrinter prints confirmed deletions as they arrive from the daemon.
type ProgressPrinter struct {
	OnProgress func(client.PruneProgress) error
	Started    bool
	out        io.Writer
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
	p.OnProgress = func(progress client.PruneProgress) error {
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
