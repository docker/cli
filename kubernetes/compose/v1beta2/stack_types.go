package v1beta2 // import "github.com/docker/cli/kubernetes/compose/v1beta2"

import (
	"encoding/json"

	composetypes "github.com/docker/cli/cli/compose/types"
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
	StackImpl
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
	result.Spec = *s.Spec.clone()
	result.Status = s.Status.clone()
	return result
}

// StackImpl contains the stack's actual fields
type StackImpl struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StackSpec    `json:"spec,omitempty"`
	Status *StackStatus `json:"status,omitempty"`
}

// StackSpec defines the desired state of Stack
type StackSpec struct {
	Stack *composetypes.Config `json:"stack,omitempty"`
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
func (s *Stack) Clone() (*Stack, error) {
	return s.clone(), nil
}

// FromCompose returns a stack from a compose config
func FromCompose(name string, composeConfig *composetypes.Config) *Stack {
	return &Stack{
		StackImpl: StackImpl{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: StackSpec{
				Stack: composeConfig,
			},
		},
	}
}

/* Do not remove me! This explicit implementation of json.Marshaler overrides
 * the default behavior of ToUnstructured(), which would otherwise convert
 * all field names to lowercase, which makes patching fail in case of update
 * conflict
 *
 */

// MarshalJSON implements the json.Marshaler interface
func (s *Stack) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.StackImpl)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (s *Stack) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &s.StackImpl)
}
