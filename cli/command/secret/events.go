package secret

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/events"
	eventtypes "github.com/docker/docker/api/types/events"
	"github.com/spf13/cobra"
)

// newSecretEventsCommand creates a new cobra.Command for `docker secret events`
func newSecretEventsCommand(dockerCli command.Cli) *cobra.Command {
	return events.NewObjectEventsCommand(dockerCli, eventtypes.SecretEventType)
}
