package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ComposeFile is the content of a stack's compose file if any
type ComposeFile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	ComposeFile       string `json:"composeFile,omitempty"`
}

func (c *ComposeFile) clone() *ComposeFile {
	if c == nil {
		return nil
	}
	res := *c
	return &res
}

// DeepCopyObject clones the ComposeFile
func (c *ComposeFile) DeepCopyObject() runtime.Object {
	return c.clone()
}
