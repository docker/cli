package v1alpha3

import (
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	conversion "k8s.io/apimachinery/pkg/conversion"
)

// Convert_v1alpha3_ServiceConfig_To_v1beta2_ServiceConfig is a wrapper around an auto-generated conversion
// nolint: golint
func Convert_v1alpha3_ServiceConfig_To_v1beta2_ServiceConfig(in *ServiceConfig, out *v1beta2.ServiceConfig, s conversion.Scope) error {
	return autoConvert_v1alpha3_ServiceConfig_To_v1beta2_ServiceConfig(in, out, s)
}
