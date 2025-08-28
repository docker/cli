package options

// Deploy holds docker stack deploy options
//
// Deprecated: this type was for internal use and will be removed in the next release.
type Deploy struct {
	Composefiles     []string
	Namespace        string
	ResolveImage     string
	SendRegistryAuth bool
	Prune            bool
	Detach           bool
	Quiet            bool
}

// Config holds docker stack config options
//
// Deprecated: this type was for internal use and will be removed in the next release.
type Config struct {
	Composefiles      []string
	SkipInterpolation bool
}

// Remove holds docker stack remove options
//
// Deprecated: this type was for internal use and will be removed in the next release.
type Remove struct {
	Namespaces []string
	Detach     bool
}
