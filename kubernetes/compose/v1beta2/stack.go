package v1beta2

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// StackList is a list of stacks
type StackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Stack `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// Stack is v1beta2's representation of a Stack
type Stack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *StackSpec   `json:"spec,omitempty"`
	Status *StackStatus `json:"status,omitempty"`
}

// DeepCopyObject clones the stack
func (s *Stack) DeepCopyObject() runtime.Object {
	return s.clone()
}

// DeepCopyObject clones the stack list
func (s *StackList) DeepCopyObject() runtime.Object {
	if s == nil {
		return nil
	}
	result := new(StackList)
	result.TypeMeta = s.TypeMeta
	result.ListMeta = s.ListMeta
	if s.Items == nil {
		return result
	}
	result.Items = make([]Stack, len(s.Items))
	for ix, s := range s.Items {
		result.Items[ix] = *s.clone()
	}
	return result
}

func (s *Stack) clone() *Stack {
	if s == nil {
		return nil
	}
	result := new(Stack)
	result.TypeMeta = s.TypeMeta
	result.ObjectMeta = s.ObjectMeta
	result.Spec = s.Spec.clone()
	result.Status = s.Status.clone()
	return result
}

// StackSpec defines the desired state of Stack
type StackSpec struct {
	Services []ServiceConfig            `json:"services,omitempty"`
	Secrets  map[string]SecretConfig    `json:"secrets,omitempty"`
	Configs  map[string]ConfigObjConfig `json:"configs,omitempty"`
}

// ServiceConfig is the configuration of one service
type ServiceConfig struct {
	Name string `json:"name,omitempty"`

	CapAdd          []string                 `json:"cap_add,omitempty"`
	CapDrop         []string                 `json:"cap_drop,omitempty"`
	Command         []string                 `json:"command,omitempty"`
	Configs         []ServiceConfigObjConfig `json:"configs,omitempty"`
	Deploy          DeployConfig             `json:"deploy,omitempty"`
	Entrypoint      []string                 `json:"entrypoint,omitempty"`
	Environment     map[string]*string       `json:"environment,omitempty"`
	ExtraHosts      []string                 `json:"extra_hosts,omitempty"`
	Hostname        string                   `json:"hostname,omitempty"`
	HealthCheck     *HealthCheckConfig       `json:"health_check,omitempty"`
	Image           string                   `json:"image,omitempty"`
	Ipc             string                   `json:"ipc,omitempty"`
	Labels          map[string]string        `json:"labels,omitempty"`
	Pid             string                   `json:"pid,omitempty"`
	Ports           []ServicePortConfig      `json:"ports,omitempty"`
	Privileged      bool                     `json:"privileged,omitempty"`
	ReadOnly        bool                     `json:"read_only,omitempty"`
	Secrets         []ServiceSecretConfig    `json:"secrets,omitempty"`
	StdinOpen       bool                     `json:"stdin_open,omitempty"`
	StopGracePeriod *time.Duration           `json:"stop_grace_period,omitempty"`
	Tmpfs           []string                 `json:"tmpfs,omitempty"`
	Tty             bool                     `json:"tty,omitempty"`
	User            *int64                   `json:"user,omitempty"`
	Volumes         []ServiceVolumeConfig    `json:"volumes,omitempty"`
	WorkingDir      string                   `json:"working_dir,omitempty"`
}

// ServicePortConfig is the port configuration for a service
type ServicePortConfig struct {
	Mode      string `json:"mode,omitempty"`
	Target    uint32 `json:"target,omitempty"`
	Published uint32 `json:"published,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
}

// FileObjectConfig is a config type for a file used by a service
type FileObjectConfig struct {
	Name     string            `json:"name,omitempty"`
	File     string            `json:"file,omitempty"`
	External External          `json:"external,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

// SecretConfig for a secret
type SecretConfig FileObjectConfig

// ConfigObjConfig is the config for the swarm "Config" object
type ConfigObjConfig FileObjectConfig

// External identifies a Volume or Network as a reference to a resource that is
// not managed, and should already exist.
// External.name is deprecated and replaced by Volume.name
type External struct {
	Name     string `json:"name,omitempty"`
	External bool   `json:"external,omitempty"`
}

// FileReferenceConfig for a reference to a swarm file object
type FileReferenceConfig struct {
	Source string  `json:"source,omitempty"`
	Target string  `json:"target,omitempty"`
	UID    string  `json:"uid,omitempty"`
	GID    string  `json:"gid,omitempty"`
	Mode   *uint32 `json:"mode,omitempty"`
}

// ServiceConfigObjConfig is the config obj configuration for a service
type ServiceConfigObjConfig FileReferenceConfig

// ServiceSecretConfig is the secret configuration for a service
type ServiceSecretConfig FileReferenceConfig

// DeployConfig is the deployment configuration for a service
type DeployConfig struct {
	Mode          string            `json:"mode,omitempty"`
	Replicas      *uint64           `json:"replicas,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	UpdateConfig  *UpdateConfig     `json:"update_config,omitempty"`
	Resources     Resources         `json:"resources,omitempty"`
	RestartPolicy *RestartPolicy    `json:"restart_policy,omitempty"`
	Placement     Placement         `json:"placement,omitempty"`
}

// UpdateConfig is the service update configuration
type UpdateConfig struct {
	Parallelism *uint64 `json:"paralellism,omitempty"`
}

// Resources the resource limits and reservations
type Resources struct {
	Limits       *Resource `json:"limits,omitempty"`
	Reservations *Resource `json:"reservations,omitempty"`
}

// Resource is a resource to be limited or reserved
type Resource struct {
	NanoCPUs    string `json:"cpus,omitempty"`
	MemoryBytes int64  `json:"memory,omitempty"`
}

// RestartPolicy is the service restart policy
type RestartPolicy struct {
	Condition string `json:"condition,omitempty"`
}

// Placement constraints for the service
type Placement struct {
	Constraints *Constraints `json:"constraints,omitempty"`
}

// Constraints lists constraints that can be set on the service
type Constraints struct {
	OperatingSystem *Constraint
	Architecture    *Constraint
	Hostname        *Constraint
	MatchLabels     map[string]Constraint
}

// Constraint defines a constraint and it's operator (== or !=)
type Constraint struct {
	Value    string
	Operator string
}

// HealthCheckConfig the healthcheck configuration for a service
type HealthCheckConfig struct {
	Test     []string       `json:"test,omitempty"`
	Timeout  *time.Duration `json:"timeout,omitempty"`
	Interval *time.Duration `json:"interval,omitempty"`
	Retries  *uint64        `json:"retries,omitempty"`
}

// ServiceVolumeConfig are references to a volume used by a service
type ServiceVolumeConfig struct {
	Type     string `json:"type,omitempty"`
	Source   string `json:"source,omitempty"`
	Target   string `json:"target,omitempty"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

func (s *StackSpec) clone() *StackSpec {
	if s == nil {
		return nil
	}
	result := *s
	return &result
}

// StackPhase is the deployment phase of a stack
type StackPhase string

// These are valid conditions of a stack.
const (
	// StackAvailable means the stack is available.
	StackAvailable StackPhase = "Available"
	// StackProgressing means the deployment is progressing.
	StackProgressing StackPhase = "Progressing"
	// StackFailure is added in a stack when one of its members fails to be created
	// or deleted.
	StackFailure StackPhase = "Failure"
)

// StackStatus defines the observed state of Stack
type StackStatus struct {
	// Current condition of the stack.
	// +optional
	Phase StackPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=StackPhase"`
	// A human readable message indicating details about the stack.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

func (s *StackStatus) clone() *StackStatus {
	if s == nil {
		return nil
	}
	result := *s
	return &result
}

// Clone clones a Stack
func (s *Stack) Clone() *Stack {
	return s.clone()
}
