package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// StackList defines a list of stacks
type StackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Stack `json:"items" protobuf:"bytes,2,rep,name=items"`
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

// Stack defines a stack object to be register in the kubernetes API
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
	Phase StackPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=StackPhase"`
	// A human readable message indicating details about the stack.
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

func (s *Stack) clone() *Stack {
	if s == nil {
		return nil
	}
	// in v1beta1, Stack has no pointer, slice or map. Plain old struct copy is ok
	result := *s
	return &result
}

// Clone implements the Cloner interface for kubernetes
func (s *Stack) Clone() *Stack {
	return s.clone()
}

// DeepCopyObject clones the stack
func (s *Stack) DeepCopyObject() runtime.Object {
	return s.clone()
}
