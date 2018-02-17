package kubernetes

import (
	"fmt"

	"github.com/docker/cli/cli/command/stack/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RunRemove is the kubernetes implementation of docker stack remove
func RunRemove(dockerCli *KubeCli, opts options.Remove) error {
	stacks, err := dockerCli.stacks()
	if err != nil {
		return err
	}
	for _, stack := range opts.Namespaces {
		fmt.Fprintf(dockerCli.Out(), "Removing stack: %s\n", stack)
		err := stacks.Delete(stack, &metav1.DeleteOptions{})
		if err != nil {
			fmt.Fprintf(dockerCli.Out(), "Failed to remove stack %s: %s\n", stack, err)
			return err
		}
	}
	return nil
}
