package v1beta2

import (
	"github.com/docker/cli/kubernetes/compose/clone"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Scale contains the current/desired replica count for services in a stack.
type Scale struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              map[string]int `json:"spec,omitempty"`
	Status            map[string]int `json:"status,omitempty"`
}

func (s *Scale) clone() *Scale {
	return &Scale{
		TypeMeta:   s.TypeMeta,
		ObjectMeta: s.ObjectMeta,
		Spec:       clone.MapOfStringToInt(s.Spec),
		Status:     clone.MapOfStringToInt(s.Status),
	}
}

// DeepCopyObject clones the scale
func (s *Scale) DeepCopyObject() runtime.Object {
	return s.clone()
}
