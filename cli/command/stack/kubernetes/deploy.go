package kubernetes

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/common"
	composeTypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type deployOptions struct {
	composefile string
	stack       string
}

func newDeployCommand(dockerCli command.Cli, kubeCli *kubeCli) *cobra.Command {
	var opts deployOptions
	cmd := &cobra.Command{
		Use:     "deploy [OPTIONS] STACK",
		Aliases: []string{"up"},
		Short:   "Deploy a new stack or update an existing stack",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stack = args[0]
			return runDeploy(dockerCli, kubeCli, opts)
		},
	}
	flags := cmd.Flags()
	common.AddComposefileFlag(&opts.composefile, flags)
	// FIXME(vdemeester) other flags ? (bundlefile, registry-auth, prune, resolve-image) ?
	return cmd
}

func runDeploy(dockerCli command.Cli, kubeCli *kubeCli, opts deployOptions) error {
	cmdOut := dockerCli.Out()
	// Check arguments
	if opts.composefile == "" {
		return errors.Errorf("Please specify a Compose file (with --compose-file).")
	}
	// Initialize clients
	stacks, err := kubeCli.Stacks()
	if err != nil {
		return err
	}
	composeClient, err := kubeCli.ComposeClient()
	if err != nil {
		return err
	}
	configMaps := composeClient.ConfigMaps()
	secrets := composeClient.Secrets()
	services := composeClient.Services()
	pods := composeClient.Pods()
	watcher := DeployWatcher{
		Pods: pods,
	}

	// Parse the compose file
	stack, cfg, err := LoadStack(opts.stack, opts.composefile)
	if err != nil {
		return err
	}

	// FIXME(vdemeester) handle warnings server-side

	if err = IsColliding(services, stack); err != nil {
		return err
	}

	if err = createFileBasedConfigMaps(stack.Name, cfg.Configs, configMaps); err != nil {
		return err
	}

	if err = createFileBasedSecrets(stack.Name, cfg.Secrets, secrets); err != nil {
		return err
	}

	if in, err := stacks.Get(stack.Name, metav1.GetOptions{}); err == nil {
		in.Spec = stack.Spec

		if _, err = stacks.Update(in); err != nil {
			return err
		}

		fmt.Printf("Stack %s was updated\n", stack.Name)
	} else {
		if _, err = stacks.Create(stack); err != nil {
			return err
		}

		fmt.Fprintf(cmdOut, "Stack %s was created\n", stack.Name)
	}

	fmt.Fprintln(cmdOut, "Waiting for the stack to be stable and running...")

	<-watcher.Watch(stack, serviceNames(cfg))

	fmt.Fprintf(cmdOut, "Stack %s is stable and running\n\n", stack.Name)
	// fmt.Fprintf(cmdOut, "Read the logs with:\n  $ %s stack logs %s\n", filepath.Base(os.Args[0]), stack.Name)

	return nil
}

// createFileBasedConfigMaps creates a Kubernetes ConfigMap for each Compose global file-based config.
func createFileBasedConfigMaps(stackName string, globalConfigs map[string]composeTypes.ConfigObjConfig, configMaps corev1.ConfigMapInterface) error {
	for name, config := range globalConfigs {
		if config.File == "" {
			continue
		}

		fileName := path.Base(config.File)
		content, err := ioutil.ReadFile(config.File)
		if err != nil {
			return err
		}

		configMap := toConfigMap(stackName, name, fileName, content)
		configMaps.Create(configMap)
	}

	return nil
}

func serviceNames(cfg *composeTypes.Config) []string {
	names := []string{}

	for _, service := range cfg.Services {
		names = append(names, service.Name)
	}

	return names
}

// createFileBasedSecrets creates a Kubernetes Secret for each Compose global file-based secret.
func createFileBasedSecrets(stackName string, globalSecrets map[string]composeTypes.SecretConfig, secrets corev1.SecretInterface) error {
	for name, secret := range globalSecrets {
		if secret.File == "" {
			continue
		}

		fileName := path.Base(secret.File)
		content, err := ioutil.ReadFile(secret.File)
		if err != nil {
			return err
		}

		secret := toSecret(stackName, name, fileName, content)
		secrets.Create(secret)
	}

	return nil
}
