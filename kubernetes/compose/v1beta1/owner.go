package v1beta1

import (
	"github.com/docker/cli/kubernetes/compose/impersonation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Owner defines the owner of a stack. It is used to impersonate the controller calls
// to kubernetes api.
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
