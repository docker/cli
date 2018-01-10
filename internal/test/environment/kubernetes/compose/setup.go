package compose

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"time"

	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/gotestyourself/gotestyourself/poll"
	"github.com/pkg/errors"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedappsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	componentKey     = "com.docker.component"
	composeComponent = "compose"
)

var labels = map[string]string{componentKey: composeComponent}

// Options holds setup options for the docker kubernetes compose component installation
type Options struct {
	Namespace              string
	Tag                    string
	ReconciliationInterval time.Duration
}

// Setup installs the Compose features on the kubernetes cluster
func Setup(w io.Writer, kubeconfigPath string, options Options) error {
	fmt.Fprintf(w, "Starting docker kube compose (%s)...\n", options.Tag)

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return errors.Wrap(err, "cannot setup compose controller")
	}

	steps := []func(*rest.Config, Options) error{
		waitForKubernetes,
		createNamespace,
		createCRD,
		createDeployment,
	}
	for _, step := range steps {
		if err := step(config, options); err != nil {
			return errors.Wrap(err, "cannot setup compose controller")
		}
	}

	t := &fakeT{}
	poll.WaitOn(t, isAPIPresent(config), poll.WithDelay(500*time.Millisecond), poll.WithTimeout(15*time.Second))
	return nil
}

func isAPIPresent(config *rest.Config) func(poll.LogT) poll.Result {
	return func(poll.LogT) poll.Result {
		clients, err := kubernetes.NewForConfig(config)
		if err != nil {
			return poll.Continue("kubernetes compose api not ready")
		}
		groups, err := clients.Discovery().ServerGroups()
		if err != nil {
			return poll.Continue("kubernetes compose api not ready")
		}
		for _, group := range groups.Groups {
			if group.Name == apiv1beta1.SchemeGroupVersion.Group {
				return poll.Success()
			}
		}

		return poll.Continue("kubernetes compose api not ready")
	}
}

func waitForKubernetes(config *rest.Config, _ Options) error {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "cannot setup compose controller")
	}
	t := time.After(10 * time.Second)
	for {
		if _, err := client.ServerVersion(); err == nil {
			break
		}
		select {
		case <-t:
			return fmt.Errorf("timed out waiting for k8s api server version")
		default:
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func createNamespace(config *rest.Config, options Options) error {
	client, err := corev1.NewForConfig(config)
	if err != nil {
		return err
	}

	_, err = client.Namespaces().Get(options.Namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	_, err = client.Namespaces().Create(&apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.Namespace,
		},
	})
	return err
}

func createCRD(config *rest.Config, _ Options) error {
	crds, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return err
	}

	_, err = crds.ApiextensionsV1beta1().CustomResourceDefinitions().Get("stacks.compose.docker.com", metav1.GetOptions{})
	if err != nil {
		err = createCustomResourceDefinition(crds, "stacks", apiv1beta1.SchemeGroupVersion, reflect.TypeOf(apiv1beta1.Stack{}), labels)
	}

	return err
}

func createCustomResourceDefinition(crds *apiextensionsclient.Clientset, plural string, groupVersion schema.GroupVersion, typeOf reflect.Type, labels map[string]string) error {
	crd, err := crds.ApiextensionsV1beta1().CustomResourceDefinitions().Create(&apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   plural + "." + groupVersion.Group,
			Labels: labels,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   groupVersion.Group,
			Version: groupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: plural,
				Kind:   typeOf.Name(),
			},
		},
	})
	if err != nil {
		return err
	}

	return wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = crds.ApiextensionsV1beta1().CustomResourceDefinitions().Get(plural+"."+groupVersion.Group, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			if cond.Type == apiextensionsv1beta1.Established && cond.Status == apiextensionsv1beta1.ConditionTrue { // TODO: replace ExtensionConditionTrue by ConditionTrue
				return true, err
			}
		}

		return false, err
	})
}

func createDeployment(config *rest.Config, options Options) error {
	apps, err := typedappsv1beta2.NewForConfig(config)
	if err != nil {
		return err
	}

	deploy := &appsv1beta2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compose",
			Namespace: options.Namespace,
			Labels:    labels,
		},
		Spec: appsv1beta2.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "compose",
							Image:           "docker/kube-compose-controller:" + options.Tag,
							ImagePullPolicy: apiv1.PullAlways,
							Args: []string{
								"--kubeconfig", "",
								"--reconciliation-interval", options.ReconciliationInterval.String(),
							},
						},
					},
				},
			},
		},
	}

	_, err = apps.Deployments(options.Namespace).Get("compose", metav1.GetOptions{})
	if err == nil {
		_, err = apps.Deployments(options.Namespace).Update(deploy)
		return err
	}
	_, err = apps.Deployments(options.Namespace).Create(deploy)
	return err
}

type fakeT struct {
	failed string
}

func (t *fakeT) Fatalf(format string, args ...interface{}) {
	t.failed = fmt.Sprintf(format, args...)
	panic("exit wait on")
}

func (t *fakeT) Log(args ...interface{}) {
	fmt.Fprint(os.Stderr, args...)
}

func (t *fakeT) Logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}
