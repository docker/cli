package builders

import (
	"github.com/docker/docker/api/types/swarm"
)

// Service creates a service with default values.
// Any number of service builder functions can be passed to augment it.
func Service(builders ...func(*swarm.Service)) *swarm.Service {
	service := &swarm.Service{}
	defaults := []func(*swarm.Service){ServiceID("serviceID"), ServiceName("defaultServiceName")}

	for _, opt := range append(defaults, builders...) {
		opt(service)
	}

	return service
}

// ServiceID sets the service ID
func ServiceID(ID string) func(*swarm.Service) {
	return func(service *swarm.Service) {
		service.ID = ID
	}
}

// ServiceName sets the service name
func ServiceName(name string) func(*swarm.Service) {
	return func(service *swarm.Service) {
		service.Spec.Annotations.Name = name
	}
}

// ServiceLabels sets the service's labels
func ServiceLabels(labels map[string]string) func(*swarm.Service) {
	return func(service *swarm.Service) {
		service.Spec.Annotations.Labels = labels
	}
}

// GlobalService sets the service to use "global" mode
func GlobalService() func(*swarm.Service) {
	return func(service *swarm.Service) {
		service.Spec.Mode = swarm.ServiceMode{Global: &swarm.GlobalService{}}
	}
}

// ReplicatedService sets the service to use "replicated" mode with the specified number of replicas
func ReplicatedService(replicas uint64) func(*swarm.Service) {
	return func(service *swarm.Service) {
		service.Spec.Mode = swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}}
		if service.ServiceStatus == nil {
			service.ServiceStatus = &swarm.ServiceStatus{}
		}
		service.ServiceStatus.DesiredTasks = replicas
	}
}

// ServiceStatus sets the services' ServiceStatus (API v1.41 and above)
func ServiceStatus(desired, running uint64) func(*swarm.Service) {
	return func(service *swarm.Service) {
		service.ServiceStatus = &swarm.ServiceStatus{
			RunningTasks: running,
			DesiredTasks: desired,
		}
	}
}

// ServiceImage sets the service's image
func ServiceImage(image string) func(*swarm.Service) {
	return func(service *swarm.Service) {
		if service.Spec.TaskTemplate.ContainerSpec == nil {
			service.Spec.TaskTemplate.ContainerSpec = &swarm.ContainerSpec{}
		}
		service.Spec.TaskTemplate.ContainerSpec.Image = image
	}
}

// ServicePort sets the service's port
func ServicePort(port swarm.PortConfig) func(*swarm.Service) {
	return func(service *swarm.Service) {
		if service.Spec.EndpointSpec == nil {
			service.Spec.EndpointSpec = &swarm.EndpointSpec{}
		}
		service.Spec.EndpointSpec.Ports = append(service.Spec.EndpointSpec.Ports, port)

		assignedPort := port
		if assignedPort.PublishedPort == 0 {
			assignedPort.PublishedPort = 30000
		}
		service.Endpoint.Ports = append(service.Endpoint.Ports, assignedPort)
	}
}
