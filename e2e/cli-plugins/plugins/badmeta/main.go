package main

// This is not a real plugin, but just returns malformated metadata
// from the subcommand and otherwise exits with failure.

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli-plugins/manager"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == manager.MetadataSubcommandName {
		fmt.Println(`{invalid-json}`)
		os.Exit(0)
	}
	os.Exit(1)
}
