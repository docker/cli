package kubernetes

import (
	"fmt"

	"github.com/docker/cli/cli/command/stack/options"
)

// RunRemove is the kubernetes implementation of docker stack remove
func RunRemove(dockerCli *KubeCli, opts options.Remove) error {
	composeClient, err := dockerCli.composeClient()
	if err != nil {
		return err
	}
	stacks, err := composeClient.Stacks(false)
	if err != nil {
		return err
	}
	for _, stack := range opts.Namespaces {
		fmt.Fprintf(dockerCli.Out(), "Removing stack: %s\n", stack)
		err := stacks.Delete(stack)
		if err != nil {
			fmt.Fprintf(dockerCli.Out(), "Failed to remove stack %s: %s\n", stack, err)
			return err
		}
	}
	return nil
}
