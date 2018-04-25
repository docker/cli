package kubernetes

import (
	"fmt"
	"io"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/schema"
	composetypes "github.com/docker/cli/cli/compose/types"
	composev1beta1 "github.com/docker/cli/kubernetes/client/clientset/typed/compose/v1beta1"
	composev1beta2 "github.com/docker/cli/kubernetes/client/clientset/typed/compose/v1beta2"
	v1beta1types "github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/docker/cli/kubernetes/labels"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// StackClient talks to a kubernetes compose component.
type StackClient interface {
	CreateOrUpdate(s stack) error
	Delete(name string) error
	Get(name string) (stack, error)
	List(opts metav1.ListOptions) ([]stack, error)
	IsColliding(servicesClient corev1.ServiceInterface, s stack) error
	FromCompose(stderr io.Writer, name string, cfg *composetypes.Config) (stack, error)
}

// stackV1Beta1 implements stackClient interface and talks to compose component v1beta1.
type stackV1Beta1 struct {
	stacks composev1beta1.StackInterface
}

func newStackV1Beta1(config *rest.Config, namespace string) (StackClient, error) {
	client, err := composev1beta1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &stackV1Beta1{stacks: client.Stacks(namespace)}, nil
}

func (s *stackV1Beta1) CreateOrUpdate(internalStack stack) error {
	// If it already exists, update the stack
	if stackBeta1, err := s.stacks.Get(internalStack.name, metav1.GetOptions{}); err == nil {
		stackBeta1.Spec.ComposeFile = internalStack.composeFile
		_, err := s.stacks.Update(stackBeta1)
		return err
	}
	// Or create it
	_, err := s.stacks.Create(stackToV1beta1(internalStack))
	return err
}

func (s *stackV1Beta1) Delete(name string) error {
	return s.stacks.Delete(name, &metav1.DeleteOptions{})
}

func (s *stackV1Beta1) Get(name string) (stack, error) {
	stackBeta1, err := s.stacks.Get(name, metav1.GetOptions{})
	if err != nil {
		return stack{}, err
	}
	return stackFromV1beta1(stackBeta1)
}

func (s *stackV1Beta1) List(opts metav1.ListOptions) ([]stack, error) {
	list, err := s.stacks.List(opts)
	if err != nil {
		return nil, err
	}
	stacks := make([]stack, len(list.Items))
	for i := range list.Items {
		stack, err := stackFromV1beta1(&list.Items[i])
		if err != nil {
			return nil, err
		}
		stacks[i] = stack
	}
	return stacks, nil
}

// IsColliding verifies that services defined in the stack collides with already deployed services
func (s *stackV1Beta1) IsColliding(servicesClient corev1.ServiceInterface, st stack) error {
	for _, srv := range st.getServices() {
		if err := verify(servicesClient, st.name, srv); err != nil {
			return err
		}
	}
	return nil
}

// verify checks wether the service is already present in kubernetes.
// If we find the service by name but it doesn't have our label or it has a different value
// than the stack name for the label, we fail (i.e. it will collide)
func verify(services corev1.ServiceInterface, stackName string, service string) error {
	svc, err := services.Get(service, metav1.GetOptions{})
	if err == nil {
		if key, ok := svc.ObjectMeta.Labels[labels.ForStackName]; ok {
			if key != stackName {
				return fmt.Errorf("service %s already present in stack named %s", service, key)
			}
			return nil
		}
		return fmt.Errorf("service %s already present in the cluster", service)
	}
	return nil
}

func (s *stackV1Beta1) FromCompose(stderr io.Writer, name string, cfg *composetypes.Config) (stack, error) {
	cfg.Version = v1beta1types.MaxComposeVersion
	st, err := fromCompose(stderr, name, cfg)
	if err != nil {
		return stack{}, err
	}
	res, err := yaml.Marshal(cfg)
	if err != nil {
		return stack{}, err
	}
	// reload the result to check that it produced a valid 3.5 compose file
	resparsedConfig, err := loader.ParseYAML(res)
	if err != nil {
		return stack{}, err
	}
	if err = schema.Validate(resparsedConfig, v1beta1types.MaxComposeVersion); err != nil {
		return stack{}, errors.Wrapf(err, "the compose yaml file is invalid with v%s", v1beta1types.MaxComposeVersion)
	}

	st.composeFile = string(res)
	return st, nil
}

// stackV1Beta2 implements stackClient interface and talks to compose component v1beta2.
type stackV1Beta2 struct {
	stacks composev1beta2.StackInterface
}

func newStackV1Beta2(config *rest.Config, namespace string) (StackClient, error) {
	client, err := composev1beta2.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &stackV1Beta2{stacks: client.Stacks(namespace)}, nil
}

func (s *stackV1Beta2) CreateOrUpdate(internalStack stack) error {
	// If it already exists, update the stack
	if stackBeta2, err := s.stacks.Get(internalStack.name, metav1.GetOptions{}); err == nil {
		stackBeta2.Spec = internalStack.spec
		_, err := s.stacks.Update(stackBeta2)
		return err
	}
	// Or create it
	_, err := s.stacks.Create(stackToV1beta2(internalStack))
	return err
}

func (s *stackV1Beta2) Delete(name string) error {
	return s.stacks.Delete(name, &metav1.DeleteOptions{})
}

func (s *stackV1Beta2) Get(name string) (stack, error) {
	stackBeta2, err := s.stacks.Get(name, metav1.GetOptions{})
	if err != nil {
		return stack{}, err
	}
	return stackFromV1beta2(stackBeta2), nil
}

func (s *stackV1Beta2) List(opts metav1.ListOptions) ([]stack, error) {
	list, err := s.stacks.List(opts)
	if err != nil {
		return nil, err
	}
	stacks := make([]stack, len(list.Items))
	for i := range list.Items {
		stacks[i] = stackFromV1beta2(&list.Items[i])
	}
	return stacks, nil
}

// IsColliding is handle server side with the compose api v1beta2, so nothing to do here
func (s *stackV1Beta2) IsColliding(servicesClient corev1.ServiceInterface, st stack) error {
	return nil
}

func (s *stackV1Beta2) FromCompose(stderr io.Writer, name string, cfg *composetypes.Config) (stack, error) {
	return fromCompose(stderr, name, cfg)
}

func fromCompose(stderr io.Writer, name string, cfg *composetypes.Config) (stack, error) {
	return stack{
		name: name,
		spec: fromComposeConfig(stderr, cfg),
	}, nil
}
