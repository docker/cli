package kubernetes

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/kubernetes/labels"
	"github.com/docker/docker/api/types/filters"
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

func fetchPods(stackName string, pods corev1.PodInterface, f filters.Args) ([]apiv1.Pod, error) {
	services := f.Get("service")
	// for existing script compatibility, support either <servicename> or <stackname>_<servicename> format
	stackNamePrefix := stackName + "_"
	for _, s := range services {
		if strings.HasPrefix(s, stackNamePrefix) {
			services = append(services, strings.TrimPrefix(s, stackNamePrefix))
		}
	}
	listOpts := metav1.ListOptions{LabelSelector: labels.SelectorForStack(stackName, services...)}
	var result []apiv1.Pod
	podsList, err := pods.List(listOpts)
	if err != nil {
		return nil, err
	}
	nodes := f.Get("node")
	for _, pod := range podsList.Items {
		if filterPod(pod, nodes) &&
			// name filter is done client side for matching partials
			f.FuzzyMatch("name", stackNamePrefix+pod.Name) {

			result = append(result, pod)
		}
	}
	return result, nil
}

func filterPod(pod apiv1.Pod, nodes []string) bool {
	if len(nodes) == 0 {
		return true
	}
	for _, name := range nodes {
		if pod.Spec.NodeName == name {
			return true
		}
	}
	return false
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

const (
	publishedServiceSuffix      = "-published"
	publishedOnRandomPortSuffix = "-random-ports"
)

func convertToServices(replicas *appsv1beta2.ReplicaSetList, daemons *appsv1beta2.DaemonSetList, services *apiv1.ServiceList) ([]swarm.Service, map[string]formatter.ServiceListInfo, error) {
	result := make([]swarm.Service, len(replicas.Items))
	infos := make(map[string]formatter.ServiceListInfo, len(replicas.Items)+len(daemons.Items))
	for i, r := range replicas.Items {
		s, err := convertToService(r.Labels[labels.ForServiceName], services, r.Spec.Template.Spec.Containers)
		if err != nil {
			return nil, nil, err
		}
		result[i] = *s
		infos[s.ID] = formatter.ServiceListInfo{
			Mode:     "replicated",
			Replicas: fmt.Sprintf("%d/%d", r.Status.AvailableReplicas, r.Status.Replicas),
		}
	}
	for _, d := range daemons.Items {
		s, err := convertToService(d.Labels[labels.ForServiceName], services, d.Spec.Template.Spec.Containers)
		if err != nil {
			return nil, nil, err
		}
		result = append(result, *s)
		infos[s.ID] = formatter.ServiceListInfo{
			Mode:     "global",
			Replicas: fmt.Sprintf("%d/%d", d.Status.NumberReady, d.Status.DesiredNumberScheduled),
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, infos, nil
}

func convertToService(serviceName string, services *apiv1.ServiceList, containers []apiv1.Container) (*swarm.Service, error) {
	serviceHeadless, err := findService(services, serviceName)
	if err != nil {
		return nil, err
	}
	stack, ok := serviceHeadless.Labels[labels.ForStackName]
	if ok {
		stack += "_"
	}
	uid := string(serviceHeadless.UID)
	s := &swarm.Service{
		ID: uid,
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: stack + serviceHeadless.Name,
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image: getContainerImage(containers),
				},
			},
		},
	}
	if serviceNodePort, err := findService(services, serviceName+publishedOnRandomPortSuffix); err == nil && serviceNodePort.Spec.Type == apiv1.ServiceTypeNodePort {
		s.Endpoint = serviceEndpoint(serviceNodePort, swarm.PortConfigPublishModeHost)
	}
	if serviceLoadBalancer, err := findService(services, serviceName+publishedServiceSuffix); err == nil && serviceLoadBalancer.Spec.Type == apiv1.ServiceTypeLoadBalancer {
		s.Endpoint = serviceEndpoint(serviceLoadBalancer, swarm.PortConfigPublishModeIngress)
	}
	return s, nil
}

func findService(services *apiv1.ServiceList, name string) (apiv1.Service, error) {
	for _, s := range services.Items {
		if s.Name == name {
			return s, nil
		}
	}
	return apiv1.Service{}, fmt.Errorf("could not find service '%s'", name)
}

func serviceEndpoint(service apiv1.Service, publishMode swarm.PortConfigPublishMode) swarm.Endpoint {
	configs := make([]swarm.PortConfig, len(service.Spec.Ports))
	for i, p := range service.Spec.Ports {
		configs[i] = swarm.PortConfig{
			PublishMode:   publishMode,
			PublishedPort: uint32(p.Port),
			TargetPort:    uint32(p.TargetPort.IntValue()),
			Protocol:      toSwarmProtocol(p.Protocol),
		}
	}
	return swarm.Endpoint{Ports: configs}
}
