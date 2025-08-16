package context

import ic "github.com/docker/cli/cli/command/internal/context"

type (
	CreateOptions = ic.CreateOptions
	UpdateOptions = ic.UpdateOptions
	RemoveOptions = ic.RemoveOptions
	ExportOptions = ic.ExportOptions
)

var NewContextCommand = ic.NewContextCommand

var (
	RunCreate = ic.RunCreate
	RunExport = ic.RunExport
	RunImport = ic.RunImport
	RunRemove = ic.RunRemove
	RunUpdate = ic.RunUpdate
	RunUse    = ic.RunUse
)
