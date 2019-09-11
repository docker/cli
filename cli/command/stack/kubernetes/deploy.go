package kubernetes

import (
	"fmt"
	"io"

	"github.com/docker/cli/cli/command/stack/options"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/cli/cli/streams"
	"github.com/morikuni/aec"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// RunDeploy is the kubernetes implementation of docker stack deploy
func RunDeploy(dockerCli *KubeCli, opts options.Deploy, cfg *composetypes.Config) error {
	cmdOut := dockerCli.Out()

	// Initialize clients
	composeClient, err := dockerCli.composeClient()
	if err != nil {
		return err
	}
	stacks, err := composeClient.Stacks(false)
	if err != nil {
		return err
	}

	stack, err := stacks.FromCompose(dockerCli.Err(), opts.Namespace, cfg)
	if err != nil {
		return err
	}

	configMaps := composeClient.ConfigMaps()
	secrets := composeClient.Secrets()
	services := composeClient.Services()

	if err := stacks.IsColliding(services, stack); err != nil {
		return err
	}

	if err := createResources(stack, stacks, configMaps, secrets); err != nil {
		return err
	}

	fmt.Fprintln(cmdOut, "Waiting for the stack to be stable and running...")
	v1beta1Cli, err := dockerCli.stacksv1beta1()
	if err != nil {
		return err
	}

	pods := composeClient.Pods()
	watcher := &deployWatcher{
		stacks: v1beta1Cli,
		pods:   pods,
	}
	statusUpdates := make(chan serviceStatus)
	displayDone := make(chan struct{})
	go func() {
		defer close(displayDone)
		display := newStatusDisplay(dockerCli.Out())
		for status := range statusUpdates {
			display.OnStatus(status)
		}
	}()

	err = watcher.Watch(stack.Name, stack.getServices(), statusUpdates)
	close(statusUpdates)
	<-displayDone
	if err != nil {
		return err
	}
	fmt.Fprintf(cmdOut, "\nStack %s is stable and running\n\n", stack.Name)
	return nil

}

func createResources(stack Stack, stacks StackClient, configMaps corev1.ConfigMapInterface, secrets corev1.SecretInterface) error {
	var childResources []childResource

	cr, err := stack.createFileBasedConfigMaps(configMaps)
	childResources = append(childResources, cr...) // make sure we collect childresources already created in case of failure
	if err != nil {
		deleteChildResources(childResources)
		return err
	}

	cr, err = stack.createFileBasedSecrets(secrets)
	childResources = append(childResources, cr...) // make sure we collect childresources already created in case of failure
	if err != nil {
		deleteChildResources(childResources)
		return err
	}

	return stacks.CreateOrUpdate(stack, childResources)
}

type statusDisplay interface {
	OnStatus(serviceStatus)
}
type metaServiceState string

const (
	metaServiceStateReady   = metaServiceState("Ready")
	metaServiceStatePending = metaServiceState("Pending")
	metaServiceStateFailed  = metaServiceState("Failed")
)

func metaStateFromStatus(status serviceStatus) metaServiceState {
	switch {
	case status.podsReady > 0:
		return metaServiceStateReady
	case status.podsPending > 0:
		return metaServiceStatePending
	default:
		return metaServiceStateFailed
	}
}

type forwardOnlyStatusDisplay struct {
	o      *streams.Out
	states map[string]metaServiceState
}

func (d *forwardOnlyStatusDisplay) OnStatus(status serviceStatus) {
	state := metaStateFromStatus(status)
	if d.states[status.name] != state {
		d.states[status.name] = state
		fmt.Fprintf(d.o, "%s: %s\n", status.name, state)
	}
}

type interactiveStatusDisplay struct {
	o        *streams.Out
	statuses []serviceStatus
}

func (d *interactiveStatusDisplay) OnStatus(status serviceStatus) {
	b := aec.EmptyBuilder
	for ix := 0; ix < len(d.statuses); ix++ {
		b = b.Up(1).EraseLine(aec.EraseModes.All)
	}
	b = b.Column(0)
	fmt.Fprint(d.o, b.ANSI)
	updated := false
	for ix, s := range d.statuses {
		if s.name == status.name {
			d.statuses[ix] = status
			s = status
			updated = true
		}
		displayInteractiveServiceStatus(s, d.o)
	}
	if !updated {
		d.statuses = append(d.statuses, status)
		displayInteractiveServiceStatus(status, d.o)
	}
}

func displayInteractiveServiceStatus(status serviceStatus, o io.Writer) {
	state := metaStateFromStatus(status)
	totalFailed := status.podsFailed + status.podsSucceeded + status.podsUnknown
	fmt.Fprintf(o, "%[1]s: %[2]s\t\t[pod status: %[3]d/%[6]d ready, %[4]d/%[6]d pending, %[5]d/%[6]d failed]\n", status.name, state,
		status.podsReady, status.podsPending, totalFailed, status.podsTotal)
}

func newStatusDisplay(o *streams.Out) statusDisplay {
	if !o.IsTerminal() {
		return &forwardOnlyStatusDisplay{o: o, states: map[string]metaServiceState{}}
	}
	return &interactiveStatusDisplay{o: o}
}
