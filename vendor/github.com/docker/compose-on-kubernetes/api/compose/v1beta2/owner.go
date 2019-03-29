package v1beta2

import (
	"github.com/docker/compose-on-kubernetes/api/compose/impersonation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Owner describes the user who created the stack
type Owner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Owner             impersonation.Config `json:"owner,omitempty"`
}

func (o *Owner) clone() *Owner {
	if o == nil {
		return nil
	}
	result := new(Owner)
	result.TypeMeta = o.TypeMeta
	result.ObjectMeta = o.ObjectMeta
	result.Owner = *result.Owner.Clone()
	return result
}

// DeepCopyObject clones the owner
func (o *Owner) DeepCopyObject() runtime.Object {
	return o.clone()
}
