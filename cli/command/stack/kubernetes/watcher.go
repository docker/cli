package kubernetes

import (
	"fmt"
	"time"

	"github.com/docker/cli/kubernetes/labels"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// DeployWatcher watches a stack deployement
type DeployWatcher struct {
	Pods corev1.PodInterface
}

// Watch watches a stuck deployement and return a chan that will holds the state of the stack
func (w DeployWatcher) Watch(name string, serviceNames []string) chan bool {
	stop := make(chan bool)

	go w.waitForPods(name, serviceNames, stop)

	return stop
}

func (w DeployWatcher) waitForPods(stackName string, serviceNames []string, stop chan bool) {
	starts := map[string]int32{}

	for {
		time.Sleep(1 * time.Second)

		list, err := w.Pods.List(metav1.ListOptions{
			LabelSelector:        labels.SelectorForStack(stackName),
			IncludeUninitialized: true,
		})
		if err != nil {
			stop <- true
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
			stop <- true
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
