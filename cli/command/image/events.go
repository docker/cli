package image

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/events"
	eventtypes "github.com/docker/docker/api/types/events"
	"github.com/spf13/cobra"
)

// newEventsCommand creates a new cobra.Command for `docker image events`
func newEventsCommand(dockerCli command.Cli) *cobra.Command {
	return events.NewObjectEventsCommand(dockerCli, eventtypes.ImageEventType)
}
