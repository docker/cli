package scheme

import (
	composev1alpha3 "github.com/docker/compose-on-kubernetes/api/compose/v1alpha3"
	composev1beta1 "github.com/docker/compose-on-kubernetes/api/compose/v1beta1"
	composev1beta2 "github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"

	// For GKE authentication
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Variables required for registration
var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

func init() {
	v1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	AddToScheme(Scheme)
}

// AddToScheme adds all types of this clientset into the given scheme. This allows composition
// of clientsets, like in:
//
//   import (
//     "k8s.io/client-go/kubernetes"
//     clientsetscheme "k8s.io/client-go/kuberentes/scheme"
//     aggregatorclientsetscheme "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/scheme"
//   )
//
//   kclientset, _ := kubernetes.NewForConfig(c)
//   aggregatorclientsetscheme.AddToScheme(clientsetscheme.Scheme)
//
// After this, RawExtensions in Kubernetes types will serialize kube-aggregator types
// correctly.
func AddToScheme(scheme *runtime.Scheme) {
	composev1alpha3.AddToScheme(scheme)
	composev1beta2.AddToScheme(scheme)
	composev1beta1.AddToScheme(scheme)

}
