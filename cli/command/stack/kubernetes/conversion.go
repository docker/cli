package kubernetes

import (
	"fmt"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/kubernetes/labels"
	"github.com/docker/docker/api/types/swarm"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Pod conversion
func podToTask(pod apiv1.Pod) swarm.Task {
	var startTime time.Time
	if pod.Status.StartTime != nil {
		startTime = (*pod.Status.StartTime).Time
	}
	task := swarm.Task{
		ID:     string(pod.UID),
		NodeID: pod.Spec.NodeName,
		Spec: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: getContainerImage(pod.Spec.Containers),
			},
		},
		DesiredState: podPhaseToState(pod.Status.Phase),
		Status: swarm.TaskStatus{
			State:     podPhaseToState(pod.Status.Phase),
			Timestamp: startTime,
			PortStatus: swarm.PortStatus{
				Ports: getPorts(pod.Spec.Containers),
			},
		},
	}

	return task
}

func podPhaseToState(phase apiv1.PodPhase) swarm.TaskState {
	switch phase {
	case apiv1.PodPending:
		return swarm.TaskStatePending
	case apiv1.PodRunning:
		return swarm.TaskStateRunning
	case apiv1.PodSucceeded:
		return swarm.TaskStateComplete
	case apiv1.PodFailed:
		return swarm.TaskStateFailed
	default:
		return swarm.TaskState("unknown")
	}
}

func toSwarmProtocol(protocol apiv1.Protocol) swarm.PortConfigProtocol {
	switch protocol {
	case apiv1.ProtocolTCP:
		return swarm.PortConfigProtocolTCP
	case apiv1.ProtocolUDP:
		return swarm.PortConfigProtocolUDP
	}
	return swarm.PortConfigProtocol("unknown")
}

func fetchPods(namespace string, pods corev1.PodInterface) ([]apiv1.Pod, error) {
	labelSelector := labels.SelectorForStack(namespace)
	podsList, err := pods.List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}
	return podsList.Items, nil
}

func getContainerImage(containers []apiv1.Container) string {
	if len(containers) == 0 {
		return ""
	}
	return containers[0].Image
}

func getPorts(containers []apiv1.Container) []swarm.PortConfig {
	if len(containers) == 0 || len(containers[0].Ports) == 0 {
		return nil
	}
	ports := make([]swarm.PortConfig, len(containers[0].Ports))
	for i, port := range containers[0].Ports {
		ports[i] = swarm.PortConfig{
			PublishedPort: uint32(port.HostPort),
			TargetPort:    uint32(port.ContainerPort),
			Protocol:      toSwarmProtocol(port.Protocol),
		}
	}
	return ports
}

type tasksBySlot []swarm.Task

func (t tasksBySlot) Len() int {
	return len(t)
}

func (t tasksBySlot) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t tasksBySlot) Less(i, j int) bool {
	// Sort by slot.
	if t[i].Slot != t[j].Slot {
		return t[i].Slot < t[j].Slot
	}

	// If same slot, sort by most recent.
	return t[j].Meta.CreatedAt.Before(t[i].CreatedAt)
}

// Replicas conversion
func replicasToServices(replicas *appsv1beta2.ReplicaSetList, services *apiv1.ServiceList) ([]swarm.Service, map[string]formatter.ServiceListInfo, error) {
	result := make([]swarm.Service, len(replicas.Items))
	infos := make(map[string]formatter.ServiceListInfo, len(replicas.Items))
	for i, r := range replicas.Items {
		service, ok := findService(services, r.Labels[labels.ForServiceName])
		if !ok {
			return nil, nil, fmt.Errorf("could not find service '%s'", r.Labels[labels.ForServiceName])
		}
		stack, ok := service.Labels[labels.ForStackName]
		if ok {
			stack += "_"
		}
		uid := string(service.UID)
		s := swarm.Service{
			ID: uid,
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: stack + service.Name,
				},
				TaskTemplate: swarm.TaskSpec{
					ContainerSpec: &swarm.ContainerSpec{
						Image: getContainerImage(r.Spec.Template.Spec.Containers),
					},
				},
			},
		}
		if service.Spec.Type == apiv1.ServiceTypeLoadBalancer {
			configs := make([]swarm.PortConfig, len(service.Spec.Ports))
			for i, p := range service.Spec.Ports {
				configs[i] = swarm.PortConfig{
					PublishMode:   swarm.PortConfigPublishModeIngress,
					PublishedPort: uint32(p.Port),
					TargetPort:    uint32(p.TargetPort.IntValue()),
					Protocol:      toSwarmProtocol(p.Protocol),
				}
			}
			s.Endpoint = swarm.Endpoint{Ports: configs}
		}
		result[i] = s
		infos[uid] = formatter.ServiceListInfo{
			Mode:     "replicated",
			Replicas: fmt.Sprintf("%d/%d", r.Status.AvailableReplicas, r.Status.Replicas),
		}
	}
	return result, infos, nil
}

func findService(services *apiv1.ServiceList, name string) (apiv1.Service, bool) {
	for _, s := range services.Items {
		if s.Name == name {
			return s, true
		}
	}
	return apiv1.Service{}, false
}
