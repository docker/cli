package system

import si "github.com/docker/cli/cli/command/internal/system"

var (
	NewSystemCommand  = si.NewSystemCommand
	NewVersionCommand = si.NewVersionCommand
	NewInfoCommand    = si.NewInfoCommand
	NewEventsCommand  = si.NewEventsCommand
	NewInspectCommand = si.NewInspectCommand
)
