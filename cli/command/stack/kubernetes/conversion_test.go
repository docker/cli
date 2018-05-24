package kubernetes

import (
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/kubernetes/labels"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryTypes "k8s.io/apimachinery/pkg/types"
	apimachineryUtil "k8s.io/apimachinery/pkg/util/intstr"
)

func TestReplicasConversionNeedsAService(t *testing.T) {
	replicas := appsv1beta2.ReplicaSetList{
		Items: []appsv1beta2.ReplicaSet{makeReplicaSet("unknown", 0, 0)},
	}
	services := apiv1.ServiceList{}
	_, _, err := convertToServices(&replicas, &appsv1beta2.DaemonSetList{}, &services)
	assert.ErrorContains(t, err, "could not find service")
}

func TestKubernetesServiceToSwarmServiceConversion(t *testing.T) {
	testCases := []struct {
		replicas         *appsv1beta2.ReplicaSetList
		services         *apiv1.ServiceList
		expectedServices []swarm.Service
		expectedListInfo map[string]formatter.ServiceListInfo
	}{
		// Match replicas with headless stack services
		{
			&appsv1beta2.ReplicaSetList{
				Items: []appsv1beta2.ReplicaSet{
					makeReplicaSet("service1", 2, 5),
					makeReplicaSet("service2", 3, 3),
				},
			},
			&apiv1.ServiceList{
				Items: []apiv1.Service{
					makeKubeService("service1", "stack", "uid1", apiv1.ServiceTypeClusterIP, nil),
					makeKubeService("service2", "stack", "uid2", apiv1.ServiceTypeClusterIP, nil),
					makeKubeService("service3", "other-stack", "uid2", apiv1.ServiceTypeClusterIP, nil),
				},
			},
			[]swarm.Service{
				makeSwarmService("stack_service1", "uid1", nil),
				makeSwarmService("stack_service2", "uid2", nil),
			},
			map[string]formatter.ServiceListInfo{
				"uid1": {Mode: "replicated", Replicas: "2/5"},
				"uid2": {Mode: "replicated", Replicas: "3/3"},
			},
		},
		// Headless service and LoadBalancer Service are tied to the same Swarm service
		{
			&appsv1beta2.ReplicaSetList{
				Items: []appsv1beta2.ReplicaSet{
					makeReplicaSet("service", 1, 1),
				},
			},
			&apiv1.ServiceList{
				Items: []apiv1.Service{
					makeKubeService("service", "stack", "uid1", apiv1.ServiceTypeClusterIP, nil),
					makeKubeService("service-published", "stack", "uid2", apiv1.ServiceTypeLoadBalancer, []apiv1.ServicePort{
						{
							Port:       80,
							TargetPort: apimachineryUtil.FromInt(80),
							Protocol:   apiv1.ProtocolTCP,
						},
					}),
				},
			},
			[]swarm.Service{
				makeSwarmService("stack_service", "uid1", []swarm.PortConfig{
					{
						PublishMode:   swarm.PortConfigPublishModeIngress,
						PublishedPort: 80,
						TargetPort:    80,
						Protocol:      swarm.PortConfigProtocolTCP,
					},
				}),
			},
			map[string]formatter.ServiceListInfo{
				"uid1": {Mode: "replicated", Replicas: "1/1"},
			},
		},
		// Headless service and NodePort Service are tied to the same Swarm service

		{
			&appsv1beta2.ReplicaSetList{
				Items: []appsv1beta2.ReplicaSet{
					makeReplicaSet("service", 1, 1),
				},
			},
			&apiv1.ServiceList{
				Items: []apiv1.Service{
					makeKubeService("service", "stack", "uid1", apiv1.ServiceTypeClusterIP, nil),
					makeKubeService("service-random-ports", "stack", "uid2", apiv1.ServiceTypeNodePort, []apiv1.ServicePort{
						{
							Port:       35666,
							TargetPort: apimachineryUtil.FromInt(80),
							Protocol:   apiv1.ProtocolTCP,
						},
					}),
				},
			},
			[]swarm.Service{
				makeSwarmService("stack_service", "uid1", []swarm.PortConfig{
					{
						PublishMode:   swarm.PortConfigPublishModeHost,
						PublishedPort: 35666,
						TargetPort:    80,
						Protocol:      swarm.PortConfigProtocolTCP,
					},
				}),
			},
			map[string]formatter.ServiceListInfo{
				"uid1": {Mode: "replicated", Replicas: "1/1"},
			},
		},
	}

	for _, tc := range testCases {
		swarmServices, listInfo, err := convertToServices(tc.replicas, &appsv1beta2.DaemonSetList{}, tc.services)
		assert.NilError(t, err)
		assert.DeepEqual(t, tc.expectedServices, swarmServices)
		assert.DeepEqual(t, tc.expectedListInfo, listInfo)
	}
}

func makeReplicaSet(service string, available, replicas int32) appsv1beta2.ReplicaSet {
	return appsv1beta2.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				labels.ForServiceName: service,
			},
		},
		Spec: appsv1beta2.ReplicaSetSpec{
			Template: apiv1.PodTemplateSpec{
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Image: "image",
						},
					},
				},
			},
		},
		Status: appsv1beta2.ReplicaSetStatus{
			AvailableReplicas: available,
			Replicas:          replicas,
		},
	}
}

func makeKubeService(service, stack, uid string, serviceType apiv1.ServiceType, ports []apiv1.ServicePort) apiv1.Service {
	return apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				labels.ForStackName: stack,
			},
			Name: service,
			UID:  apimachineryTypes.UID(uid),
		},
		Spec: apiv1.ServiceSpec{
			Type:  serviceType,
			Ports: ports,
		},
	}
}

func makeSwarmService(service, id string, ports []swarm.PortConfig) swarm.Service {
	return swarm.Service{
		ID: id,
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: service,
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image: "image",
				},
			},
		},
		Endpoint: swarm.Endpoint{
			Ports: ports,
		},
	}
}
