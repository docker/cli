package options

import "github.com/docker/cli/opts"

// Deploy holds docker stack deploy options
type Deploy struct {
	Bundlefile       string
	Composefiles     []string
	Namespace        string
	ResolveImage     string
	SendRegistryAuth bool
	Prune            bool
}

// List holds docker stack ls options
type List struct {
	Format        string
	AllNamespaces bool
	Namespaces    []string
}

// PS holds docker stack ps options
type PS struct {
	Filter    opts.FilterOpt
	NoTrunc   bool
	Namespace string
	NoResolve bool
	Quiet     bool
	Format    string
}

// Remove holds docker stack remove options
type Remove struct {
	Namespaces []string
}

// Services holds docker stack services options
type Services struct {
	Quiet     bool
	Format    string
	Filter    opts.FilterOpt
	Namespace string
}
