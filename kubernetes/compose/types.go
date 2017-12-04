package compose

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImpersonationConfig holds information use to impersonate calls from the compose controller
type ImpersonationConfig struct {
	// UserName is the username to impersonate on each request.
	UserName string
	// Groups are the groups to impersonate on each request.
	Groups []string
	// Extra is a free-form field which can be used to link some authentication information
	// to authorization information.  This field allows you to impersonate it.
	Extra map[string][]string
}

// Stack defines a stack object to be register in the kubernetes API
// +genclient=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Stack struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec   StackSpec
	Status StackStatus
}

// StackStatus defines the observed state of Stack
type StackStatus struct {
	Phase   StackPhase
	Message string
}

// StackSpec defines the desired state of Stack
type StackSpec struct {
	ComposeFile string
	Owner       ImpersonationConfig
}

// StackPhase defines the status phase in which the stack is.
type StackPhase string

// These are valid conditions of a stack.
const (
	// Available means the stack is available.
	StackAvailable StackPhase = "Available"
	// Progressing means the deployment is progressing.
	StackProgressing StackPhase = "Progressing"
	// StackFailure is added in a stack when one of its members fails to be created
	// or deleted.
	StackFailure StackPhase = "Failure"
)

// StackList defines a list of stacks
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StackList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Stack
}

// Owner defines the owner of a stack. It is used to impersonate the controller calls
// to kubernetes api.
type Owner struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Owner ImpersonationConfig
}

// OwnerList defines a list of owner.
type OwnerList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Owner
}

// FIXME(vdemeester) are those necessary ??

// NewStatus is newStatus
func (Stack) NewStatus() interface{} {
	return StackStatus{}
}

// GetStatus returns the status
func (pc *Stack) GetStatus() interface{} {
	return pc.Status
}

// SetStatus sets the status
func (pc *Stack) SetStatus(s interface{}) {
	pc.Status = s.(StackStatus)
}

// GetSpec returns the spec
func (pc *Stack) GetSpec() interface{} {
	return pc.Spec
}

// SetSpec sets the spec
func (pc *Stack) SetSpec(s interface{}) {
	pc.Spec = s.(StackSpec)
}

// GetObjectMeta returns the ObjectMeta
func (pc *Stack) GetObjectMeta() *metav1.ObjectMeta {
	return &pc.ObjectMeta
}

// SetGeneration sets the Generation
func (pc *Stack) SetGeneration(generation int64) {
	pc.ObjectMeta.Generation = generation
}

// GetGeneration returns the Generation
func (pc Stack) GetGeneration() int64 {
	return pc.ObjectMeta.Generation
}
