package manager

import (
	"github.com/docker/cli/cli-plugins/metadata"
)

const (
	// NamePrefix is the prefix required on all plugin binary names
	NamePrefix = metadata.NamePrefix

	// MetadataSubcommandName is the name of the plugin subcommand
	// which must be supported by every plugin and returns the
	// plugin metadata.
	MetadataSubcommandName = metadata.MetadataSubcommandName

	// HookSubcommandName is the name of the plugin subcommand
	// which must be implemented by plugins declaring support
	// for hooks in their metadata.
	HookSubcommandName = metadata.HookSubcommandName
)

// Metadata provided by the plugin.
type Metadata = metadata.Metadata
