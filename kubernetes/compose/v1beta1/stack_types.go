package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// StackList defines a list of stacks
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Stack `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Stack defines a stack object to be register in the kubernetes API
// +k8s:openapi-gen=true
// +resource:path=stacks,strategy=StackStrategy
// +subresource:request=Owner,path=owner,rest=OwnerStackREST
type Stack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StackSpec   `json:"spec,omitempty"`
	Status StackStatus `json:"status,omitempty"`
}

// StackSpec defines the desired state of Stack
type StackSpec struct {
	ComposeFile string `json:"composeFile,omitempty"`
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

// StackStatus defines the observed state of Stack
type StackStatus struct {
	// Current condition of the stack.
	// +optional
	Phase StackPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=StackPhase"`
	// A human readable message indicating details about the stack.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

// Clone implements the Cloner interface for kubernetes
func (s *Stack) Clone() (*Stack, error) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		return nil, err
	}
	return s.DeepCopy(), nil
}
