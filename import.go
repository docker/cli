// This is only used to define imports we need for doc generation.

//go:build never
// +build never

package cli

import (
	// Used for md and yaml doc generation.
	_ "github.com/docker/cli-docs-tool"

	// Used for man page generation.
	_ "github.com/cpuguy83/go-md2man/v2"
	_ "github.com/spf13/cobra"
	_ "github.com/spf13/cobra/doc"
	_ "github.com/spf13/pflag"
)
