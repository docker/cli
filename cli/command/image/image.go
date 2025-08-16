package image

import ii "github.com/docker/cli/cli/command/internal/image"

var (
	NewBuildCommand  = ii.NewBuildCommand
	NewPullCommand   = ii.NewPullCommand
	NewPushCommand   = ii.NewPushCommand
	NewImagesCommand = ii.NewImagesCommand

	// Management Commands
	NewImageCommand = ii.NewImageCommand

	// Legacy Commands
	NewHistoryCommand = ii.NewHistoryCommand
	NewImportCommand  = ii.NewImportCommand
	NewLoadCommand    = ii.NewLoadCommand
	NewRemoveCommand  = ii.NewRemoveCommand
	NewSaveCommand    = ii.NewSaveCommand
	NewTagCommand     = ii.NewTagCommand
)
