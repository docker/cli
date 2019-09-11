package v1beta2

import api "github.com/docker/compose-on-kubernetes/api/compose/v1beta2"

// StackList is a list of stacks
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.StackList instead
type StackList = api.StackList

// Stack is v1beta2's representation of a Stack
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.Stack instead
type Stack = api.Stack

// StackSpec defines the desired state of Stack
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.StackSpec instead
type StackSpec = api.StackSpec

// ServiceConfig is the configuration of one service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.ServiceConfig instead
type ServiceConfig = api.ServiceConfig

// ServicePortConfig is the port configuration for a service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.ServicePortConfig instead
type ServicePortConfig = api.ServicePortConfig

// FileObjectConfig is a config type for a file used by a service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.FileObjectConfig instead
type FileObjectConfig = api.FileObjectConfig

// SecretConfig for a secret
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.SecretConfig instead
type SecretConfig = api.SecretConfig

// ConfigObjConfig is the config for the swarm "Config" object
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.ConfigObjConfig instead
type ConfigObjConfig = api.ConfigObjConfig

// External identifies a Volume or Network as a reference to a resource that is
// not managed, and should already exist.
// External.name is deprecated and replaced by Volume.name
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.External instead
type External = api.External

// FileReferenceConfig for a reference to a swarm file object
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.FileReferenceConfig instead
type FileReferenceConfig = api.FileReferenceConfig

// ServiceConfigObjConfig is the config obj configuration for a service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.ServiceConfigObjConfig instead
type ServiceConfigObjConfig = api.ServiceConfigObjConfig

// ServiceSecretConfig is the secret configuration for a service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.ServiceSecretConfig instead
type ServiceSecretConfig = api.ServiceSecretConfig

// DeployConfig is the deployment configuration for a service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.DeployConfig instead
type DeployConfig = api.DeployConfig

// UpdateConfig is the service update configuration
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.UpdateConfig instead
type UpdateConfig = api.UpdateConfig

// Resources the resource limits and reservations
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.Resources instead
type Resources = api.Resources

// Resource is a resource to be limited or reserved
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.Resource instead
type Resource = api.Resource

// RestartPolicy is the service restart policy
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.RestartPolicy instead
type RestartPolicy = api.RestartPolicy

// Placement constraints for the service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.Placement instead
type Placement = api.Placement

// Constraints lists constraints that can be set on the service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.Constraints instead
type Constraints = api.Constraints

// Constraint defines a constraint and it's operator (== or !=)
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.Constraint instead
type Constraint = api.Constraint

// HealthCheckConfig the healthcheck configuration for a service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.HealthCheckConfig instead
type HealthCheckConfig = api.HealthCheckConfig

// ServiceVolumeConfig are references to a volume used by a service
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.ServiceVolumeConfig instead
type ServiceVolumeConfig = api.ServiceVolumeConfig

// StackPhase is the deployment phase of a stack
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.StackPhase instead
type StackPhase = api.StackPhase

// These are valid conditions of a stack.
const (
	// StackAvailable means the stack is available.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.StackAvailable instead
	StackAvailable StackPhase = api.StackAvailable
	// StackProgressing means the deployment is progressing.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.StackProgressing instead
	StackProgressing StackPhase = api.StackProgressing
	// StackFailure is added in a stack when one of its members fails to be created
	// or deleted.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.StackFailure instead
	StackFailure StackPhase = api.StackFailure
)

// StackStatus defines the observed state of Stack
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.StackStatus instead
type StackStatus = api.StackStatus
