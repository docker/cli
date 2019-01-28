// Api versions allow the api contract for a resource to be changed while keeping
// backward compatibility by support multiple concurrent versions
// of the same resource

// Package v1alpha3 is the current in dev version of the stack, containing evolution on top of v1beta2 structured spec
// +k8s:openapi-gen=true
// +k8s:conversion-gen=github.com/docker/compose-on-kubernetes/api/compose/v1beta2
package v1alpha3
