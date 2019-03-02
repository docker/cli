package kubernetes

import (
	"fmt"

	"github.com/docker/cli/cli/command/stack/options"
	"github.com/pkg/errors"
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
		if err := stacks.Delete(stack); err != nil {
			return errors.Wrapf(err, "Failed to remove stack %s", stack)
		}
	}
	return nil
}
