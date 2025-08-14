package builder

import ib "github.com/docker/cli/cli/command/internal/builder"

var (
	NewPruneCommand = ib.NewPruneCommand
	// Management Commands
	NewBuilderCommand  = ib.NewBuilderCommand
	NewBakeStubCommand = ib.NewBakeStubCommand
)
