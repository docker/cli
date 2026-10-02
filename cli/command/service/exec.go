package service

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/connhelper"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type execOptions struct {
	container.ExecOptions

	service string
	taskID  string
	sshUser string
	sshOpts []string
}

// newExecCommand creates a new cobra.Command for "docker service exec".
func newExecCommand(dockerCLI command.Cli) *cobra.Command {
	options := execOptions{ExecOptions: container.NewExecOptions()}

	cmd := &cobra.Command{
		Use:   "exec [OPTIONS] SERVICE COMMAND [ARG...]",
		Short: "Execute a command in a running task of a service, on whichever node it runs",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.service = args[0]
			options.Command = args[1:]
			return runExec(cmd.Context(), dockerCLI, options)
		},
		Annotations:           map[string]string{"version": "1.29"},
		ValidArgsFunction:     completeServiceNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)
	flags.BoolVarP(&options.Interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	flags.BoolVarP(&options.TTY, "tty", "t", false, "Allocate a pseudo-TTY")
	flags.StringVarP(&options.User, "user", "u", "", `Username or UID (format: "<name|uid>[:<group|gid>]")`)
	flags.StringVarP(&options.Workdir, "workdir", "w", "", "Working directory inside the container")
	flags.VarP(&options.Env, "env", "e", "Set environment variables")
	flags.StringVar(&options.DetachKeys, "detach-keys", "", "Override the key sequence for detaching a container")
	flags.StringVar(&options.taskID, "task-id", "", "Exec into a specific task instead of the first running one")
	flags.StringVar(&options.sshUser, "ssh-user", "", "Username for the SSH connection to the node")
	flags.StringSliceVar(&options.sshOpts, "ssh-option", nil, `Additional flags passed to ssh (e.g. "-J bastion")`)
	return cmd
}

// runExec is the swarm implementation of docker service exec. It resolves the
// service to a running task, then runs the exec either directly (if the task
// runs on the current node) or through an SSH tunnel to the node running the
// task, using the same connection helper as DOCKER_HOST=ssh://.
func runExec(ctx context.Context, dockerCLI command.Cli, options execOptions) error {
	apiClient := dockerCLI.Client()

	service, err := apiClient.ServiceInspect(ctx, options.service, client.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	tasks, err := apiClient.TaskList(ctx, client.TaskListOptions{
		Filters: make(client.Filters).
			Add("service", service.Service.ID).
			Add("desired-state", string(swarm.TaskStateRunning)),
	})
	if err != nil {
		return err
	}
	task, err := pickTask(tasks.Items, options.taskID)
	if err != nil {
		return fmt.Errorf("service %s: %w", options.service, err)
	}

	info, err := apiClient.Info(ctx, client.InfoOptions{})
	if err != nil {
		return err
	}

	containerID := task.Status.ContainerStatus.ContainerID
	if info.Info.Swarm.NodeID == task.NodeID {
		// The task runs on the node we are already talking to: plain exec.
		return container.RunExec(ctx, dockerCLI, containerID, options.ExecOptions)
	}

	node, err := apiClient.NodeInspect(ctx, task.NodeID, client.NodeInspectOptions{})
	if err != nil {
		return err
	}

	remoteClient, err := newNodeClient(nodeSSHHost(node.Node, options.sshUser), options.sshOpts)
	if err != nil {
		return err
	}
	defer remoteClient.Close()

	fmt.Fprintf(dockerCLI.Err(), "executing on node %s (%s)\n", node.Node.Description.Hostname, task.NodeID)
	return container.RunExec(ctx, &nodeCli{Cli: dockerCLI, client: remoteClient}, containerID, options.ExecOptions)
}

// pickTask returns the task to exec into: the one matching taskID if given
// (which must be running), otherwise the first running task.
func pickTask(tasks []swarm.Task, taskID string) (swarm.Task, error) {
	for _, t := range tasks {
		if taskID != "" {
			if t.ID != taskID {
				continue
			}
			if t.Status.State != swarm.TaskStateRunning || t.Status.ContainerStatus == nil {
				return swarm.Task{}, fmt.Errorf("task %s is not running (state: %s)", taskID, t.Status.State)
			}
			return t, nil
		}
		if t.Status.State == swarm.TaskStateRunning && t.Status.ContainerStatus != nil {
			return t, nil
		}
	}
	if taskID != "" {
		return swarm.Task{}, fmt.Errorf("task %s not found among tasks of the service", taskID)
	}
	return swarm.Task{}, fmt.Errorf("no running task found")
}

// nodeSSHHost returns the ssh:// URL used to reach the node running the
// task. It prefers the address advertised in the node status, falling back
// to the node hostname (relying on DNS) when it is unspecified.
func nodeSSHHost(node swarm.Node, sshUser string) string {
	addr := node.Status.Addr
	if addr == "" || addr == "0.0.0.0" {
		addr = node.Description.Hostname
	}
	if sshUser != "" {
		return "ssh://" + sshUser + "@" + addr
	}
	return "ssh://" + addr
}

// newNodeClient returns an API client connected to the docker engine on the
// given node through an SSH tunnel ("docker system dial-stdio"), like
// DOCKER_HOST=ssh:// does.
func newNodeClient(sshHost string, sshFlags []string) (client.APIClient, error) {
	helper, err := connhelper.GetConnectionHelperWithSSHOpts(sshHost, sshFlags)
	if err != nil {
		return nil, err
	}
	return client.New(
		client.WithHost(helper.Host),
		client.WithDialContext(helper.Dialer),
		client.WithAPIVersionNegotiation(),
	)
}

// nodeCli decorates a command.Cli, substituting the API client with one
// connected to the node running the task, so that the regular exec plumbing
// (TTY, resize, exit code) is reused as-is.
type nodeCli struct {
	command.Cli
	client client.APIClient
}

func (c *nodeCli) Client() client.APIClient {
	return c.client
}
