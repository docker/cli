package manager

const (
	// NamePrefix is the prefix required on all plugin binary names
	NamePrefix = "docker-"

	// MetadataSubcommandName is the name of the plugin subcommand
	// which must be supported by every plugin and returns the
	// plugin metadata.
	MetadataSubcommandName = "docker-cli-plugin-metadata"
)

// Metadata provided by the plugin
type Metadata struct {
	// SchemaVersion describes the version of this struct. Mandatory, must be "0.1.0"
	SchemaVersion string
	// Vendor is the name of the plugin vendor. Mandatory
	Vendor string
	// Version is the optional version of this plugin.
	Version string
	// ShortDescription should be suitable for a single line help message.
	ShortDescription string
	// URL is a pointer to the plugin's homepage.
	URL string
}
