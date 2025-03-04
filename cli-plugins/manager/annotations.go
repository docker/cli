package manager

import "github.com/docker/cli/cli-plugins/metadata"

const (
	// CommandAnnotationPlugin is added to every stub command added by
	// AddPluginCommandStubs with the value "true" and so can be
	// used to distinguish plugin stubs from regular commands.
	CommandAnnotationPlugin = metadata.CommandAnnotationPlugin

	// CommandAnnotationPluginVendor is added to every stub command
	// added by AddPluginCommandStubs and contains the vendor of
	// that plugin.
	CommandAnnotationPluginVendor = metadata.CommandAnnotationPluginVendor

	// CommandAnnotationPluginVersion is added to every stub command
	// added by AddPluginCommandStubs and contains the version of
	// that plugin.
	CommandAnnotationPluginVersion = metadata.CommandAnnotationPluginVersion

	// CommandAnnotationPluginInvalid is added to any stub command
	// added by AddPluginCommandStubs for an invalid command (that
	// is, one which failed it's candidate test) and contains the
	// reason for the failure.
	CommandAnnotationPluginInvalid = metadata.CommandAnnotationPluginInvalid

	// CommandAnnotationPluginCommandPath is added to overwrite the
	// command path for a plugin invocation.
	CommandAnnotationPluginCommandPath = metadata.CommandAnnotationPluginCommandPath
)
