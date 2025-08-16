package container

import (
	ic "github.com/docker/cli/cli/command/internal/container"
)

var (
	NewRunCommand       = ic.NewRunCommand
	NewExecCommand      = ic.NewExecCommand
	NewPsCommand        = ic.NewPsCommand
	NewContainerCommand = ic.NewContainerCommand

	// Legacy commands

	NewAttachCommand  = ic.NewAttachCommand
	NewCommitCommand  = ic.NewCommitCommand
	NewCopyCommand    = ic.NewCopyCommand
	NewCreateCommand  = ic.NewCreateCommand
	NewDiffCommand    = ic.NewDiffCommand
	NewExportCommand  = ic.NewExportCommand
	NewKillCommand    = ic.NewKillCommand
	NewLogsCommand    = ic.NewLogsCommand
	NewPauseCommand   = ic.NewPauseCommand
	NewPortCommand    = ic.NewPortCommand
	NewRenameCommand  = ic.NewRenameCommand
	NewRestartCommand = ic.NewRestartCommand
	NewRmCommand      = ic.NewRmCommand
	NewStartCommand   = ic.NewStartCommand
	NewStatsCommand   = ic.NewStatsCommand
	NewStopCommand    = ic.NewStopCommand
	NewTopCommand     = ic.NewTopCommand
	NewUnpauseCommand = ic.NewUnpauseCommand
	NewUpdateCommand  = ic.NewUpdateCommand
	NewWaitCommand    = ic.NewWaitCommand
)
