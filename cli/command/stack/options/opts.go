package options

import "github.com/docker/cli/opts"

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

// List holds docker stack ls options
//
// Deprecated: this type was for internal use and will be removed in the next release.
type List struct {
	Format        string
	AllNamespaces bool
}

// PS holds docker stack ps options
//
// Deprecated: this type was for internal use and will be removed in the next release.
type PS struct {
	Filter    opts.FilterOpt
	NoTrunc   bool
	Namespace string
	NoResolve bool
	Quiet     bool
	Format    string
}

// Remove holds docker stack remove options
//
// Deprecated: this type was for internal use and will be removed in the next release.
type Remove struct {
	Namespaces []string
	Detach     bool
}

// Services holds docker stack services options
//
// Deprecated: this type was for internal use and will be removed in the next release.
type Services struct {
	Quiet     bool
	Format    string
	Filter    opts.FilterOpt
	Namespace string
}
