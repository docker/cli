// Package v1beta1 holds the v1beta1 versions of our stack structures.
// API versions allow the api contract for a resource to be changed while keeping
// backward compatibility by support multiple concurrent versions
// of the same resource
//
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=github.com/docker/cli/kubernetes/compose
// +k8s:defaulter-gen=TypeMeta
// +groupName=compose.docker.com
package v1beta1 // import "github.com/docker/cli/kubernetes/compose/v1beta1"
