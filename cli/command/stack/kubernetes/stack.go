package kubernetes

import (
	"fmt"

	composeTypes "github.com/docker/cli/cli/compose/types"
	composev1beta1 "github.com/docker/cli/kubernetes/client/clientset/typed/compose/v1beta1"
	composev1beta2 "github.com/docker/cli/kubernetes/client/clientset/typed/compose/v1beta2"
	"github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/docker/cli/kubernetes/compose/v1beta2"
	"github.com/docker/cli/kubernetes/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// stackClient talks to a kubernetes compose component.
type stackClient interface {
	CreateOrUpdate(s stack) error
	Delete(name string) error
	Get(name string) (stack, error)
	List(opts metav1.ListOptions) ([]stack, error)
	IsColliding(servicesClient corev1.ServiceInterface, name string, services []string) error
}

// stack is the main type used by stack commands so they remain independant from kubernetes compose component version.
type stack struct {
	name        string
	composeFile string
	config      *composeTypes.Config
}

// stackV1Beta1 implements stackClient interface and talks to compose component v1beta1.
type stackV1Beta1 struct {
	stacks composev1beta1.StackInterface
}

func (c *KubeCli) newStackV1Beta1() (stackClient, error) {
	client, err := composev1beta1.NewForConfig(c.KubeConfig)
	if err != nil {
		return nil, err
	}
	return &stackV1Beta1{stacks: client.Stacks(c.KubeNamespace)}, nil
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
func (s *stackV1Beta1) IsColliding(servicesClient corev1.ServiceInterface, stack string, services []string) error {
	for _, srv := range services {
		if err := verify(servicesClient, stack, srv); err != nil {
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

// stackV1Beta2 implements stackClient interface and talks to compose component v1beta2.
type stackV1Beta2 struct {
	stacks composev1beta2.StackInterface
}

func (c *KubeCli) newStackV1Beta2() (stackClient, error) {
	client, err := composev1beta2.NewForConfig(c.KubeConfig)
	if err != nil {
		return nil, err
	}
	return &stackV1Beta2{stacks: client.Stacks(c.KubeNamespace)}, nil
}

func (s *stackV1Beta2) CreateOrUpdate(internalStack stack) error {
	// If it already exists, update the stack
	if stackBeta2, err := s.stacks.Get(internalStack.name, metav1.GetOptions{}); err == nil {
		stackBeta2.Spec.Stack = internalStack.config
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
func (s *stackV1Beta2) IsColliding(servicesClient corev1.ServiceInterface, stack string, services []string) error {
	return nil
}

// Conversions from internal stack to different stack compose component versions.
func stackFromV1beta1(in *v1beta1.Stack) (stack, error) {
	cfg, err := loadStackData(in.Spec.ComposeFile)
	if err != nil {
		return stack{}, err
	}
	return stack{
		name:        in.ObjectMeta.Name,
		composeFile: in.Spec.ComposeFile,
		config:      cfg,
	}, nil
}

func stackToV1beta1(s stack) *v1beta1.Stack {
	return &v1beta1.Stack{
		v1beta1.StackImpl{
			ObjectMeta: metav1.ObjectMeta{
				Name: s.name,
			},
			Spec: v1beta1.StackSpec{
				ComposeFile: s.composeFile,
			},
		},
	}
}

func stackFromV1beta2(in *v1beta2.Stack) stack {
	return stack{
		name:   in.ObjectMeta.Name,
		config: in.Spec.Stack,
	}
}

func stackToV1beta2(s stack) *v1beta2.Stack {
	return &v1beta2.Stack{
		v1beta2.StackImpl{
			ObjectMeta: metav1.ObjectMeta{
				Name: s.name,
			},
			Spec: v1beta2.StackSpec{
				Stack: s.config,
			},
		},
	}
}
