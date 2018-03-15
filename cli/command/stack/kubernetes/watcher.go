package kubernetes

import (
	"fmt"
	"io"
	"time"

	composev1beta1 "github.com/docker/cli/kubernetes/client/clientset_generated/clientset/typed/compose/v1beta1"
	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/docker/cli/kubernetes/labels"
	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// DeployWatcher watches a stack deployement
type deployWatcher struct {
	pods   corev1.PodInterface
	stacks composev1beta1.StackInterface
	out    io.Writer
}

// Watch watches a stuck deployement and return a chan that will holds the state of the stack
func (w *deployWatcher) Watch(stack *apiv1beta1.Stack, serviceNames []string) error {
	err := make(chan error)

	go w.watchStackStatus(stack.Name, err)
	go w.waitForPods(stack.Name, serviceNames, err)

	return <-err
}

func (w *deployWatcher) watchStackStatus(stackname string, e chan error) {

	watcher, err := w.stacks.Watch(metav1.ListOptions{
		LabelSelector: "com.docker.stack.namespace=" + stackname,
	})
	if err != nil {
		e <- err
		return
	}

	for {
		select {
		case ev := <-watcher.ResultChan():
			if ev.Type != watch.Added && ev.Type != watch.Modified {
				continue
			}
			stack := ev.Object.(*apiv1beta1.Stack)
			if stack.Status.Phase == apiv1beta1.StackFailure {
				e <- errors.Errorf("stack %s failed with status %s", stackname, stack.Status.Phase)
				return
			}
		case <-e:
			return
		}
	}
}

func (w *deployWatcher) waitForPods(stackName string, serviceNames []string, e chan error) {
	starts := map[string]int32{}
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()

	for {
		list, err := w.pods.List(metav1.ListOptions{
			LabelSelector:        labels.SelectorForStack(stackName),
			IncludeUninitialized: true,
		})
		if err != nil {
			e <- err
			return
		}

		for i := range list.Items {
			pod := list.Items[i]
			if pod.Status.Phase != apiv1.PodRunning {
				continue
			}

			startCount := startCount(pod)
			serviceName := pod.Labels[labels.ForServiceName]
			if startCount != starts[serviceName] {
				if startCount == 1 {
					fmt.Printf(" - Service %s has one container running\n", serviceName)
				} else {
					fmt.Printf(" - Service %s was restarted %d %s\n", serviceName, startCount-1, timeTimes(startCount-1))
				}

				starts[serviceName] = startCount
			}
		}

		if allReady(list.Items, serviceNames) {
			e <- nil
			return
		}
		select {
		case <-t.C:
			continue
		case <-e:
			return
		}
	}
}

func startCount(pod apiv1.Pod) int32 {
	restart := int32(0)

	for _, status := range pod.Status.ContainerStatuses {
		restart += status.RestartCount
	}

	return 1 + restart
}

func allReady(pods []apiv1.Pod, serviceNames []string) bool {
	serviceUp := map[string]bool{}

	for _, pod := range pods {
		if time.Since(pod.GetCreationTimestamp().Time) < 10*time.Second {
			return false
		}

		ready := false
		for _, cond := range pod.Status.Conditions {
			if cond.Type == apiv1.PodReady && cond.Status == apiv1.ConditionTrue {
				ready = true
			}
		}

		if !ready {
			return false
		}

		serviceName := pod.Labels[labels.ForServiceName]
		serviceUp[serviceName] = true
	}

	for _, serviceName := range serviceNames {
		if !serviceUp[serviceName] {
			return false
		}
	}

	return true
}

func timeTimes(n int32) string {
	if n == 1 {
		return "time"
	}

	return "times"
}
