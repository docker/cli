package kubernetes

import (
	"errors"
	"testing"

	composev1beta1 "github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1"
	composev1beta2 "github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta2"
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta1"
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	"gotest.tools/assert"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
)

func testStack() Stack {
	return Stack{
		Name:      "test",
		Namespace: "test",
		ComposeFile: `version: "3.3"
services: 
  test:
    image: nginx
secrets:
  test:
    file: testdata/secret
configs:
  test:
    file: testdata/config
`,
		Spec: &v1beta2.StackSpec{
			Configs: map[string]v1beta2.ConfigObjConfig{
				"test": {Name: "test", File: "testdata/config"},
			},
			Secrets: map[string]v1beta2.SecretConfig{
				"test": {Name: "test", File: "testdata/secret"},
			},
		},
	}
}

func TestCreateChildResourcesV1Beta1(t *testing.T) {
	k8sclientSet := fake.NewSimpleClientset()
	stack := testStack()
	configs := k8sclientSet.CoreV1().ConfigMaps("test")
	secrets := k8sclientSet.CoreV1().Secrets("test")
	assert.NilError(t, createResources(
		stack,
		&stackV1Beta1{stacks: &fakeV1beta1Client{}},
		configs,
		secrets))
	c, err := configs.Get("test", metav1.GetOptions{})
	assert.NilError(t, err)
	checkOwnerReferences(t, c.ObjectMeta, "test", v1beta1.SchemeGroupVersion.String())
	s, err := secrets.Get("test", metav1.GetOptions{})
	assert.NilError(t, err)
	checkOwnerReferences(t, s.ObjectMeta, "test", v1beta1.SchemeGroupVersion.String())
}

func checkOwnerReferences(t *testing.T, objMeta metav1.ObjectMeta, stackName, stackVersion string) {
	t.Helper()
	assert.Equal(t, len(objMeta.OwnerReferences), 1)
	assert.Equal(t, objMeta.OwnerReferences[0].Name, stackName)
	assert.Equal(t, objMeta.OwnerReferences[0].Kind, "Stack")
	assert.Equal(t, objMeta.OwnerReferences[0].APIVersion, stackVersion)
}

func TestCreateChildResourcesV1Beta2(t *testing.T) {
	k8sclientSet := fake.NewSimpleClientset()
	stack := testStack()
	configs := k8sclientSet.CoreV1().ConfigMaps("test")
	secrets := k8sclientSet.CoreV1().Secrets("test")
	assert.NilError(t, createResources(
		stack,
		&stackV1Beta2{stacks: &fakeV1beta2Client{}},
		configs,
		secrets))
	c, err := configs.Get("test", metav1.GetOptions{})
	assert.NilError(t, err)
	checkOwnerReferences(t, c.ObjectMeta, "test", v1beta2.SchemeGroupVersion.String())
	s, err := secrets.Get("test", metav1.GetOptions{})
	assert.NilError(t, err)
	checkOwnerReferences(t, s.ObjectMeta, "test", v1beta2.SchemeGroupVersion.String())
}

func TestCreateChildResourcesWithStackCreationErrorV1Beta1(t *testing.T) {
	k8sclientSet := fake.NewSimpleClientset()
	stack := testStack()
	configs := k8sclientSet.CoreV1().ConfigMaps("test")
	secrets := k8sclientSet.CoreV1().Secrets("test")
	err := createResources(
		stack,
		&stackV1Beta1{stacks: &fakeV1beta1Client{errorOnCreate: true}},
		configs,
		secrets)
	assert.Error(t, err, "some error")
	_, err = configs.Get("test", metav1.GetOptions{})
	assert.Check(t, kerrors.IsNotFound(err))
	_, err = secrets.Get("test", metav1.GetOptions{})
	assert.Check(t, kerrors.IsNotFound(err))
}

func TestCreateChildResourcesWithStackCreationErrorV1Beta2(t *testing.T) {
	k8sclientSet := fake.NewSimpleClientset()
	stack := testStack()
	configs := k8sclientSet.CoreV1().ConfigMaps("test")
	secrets := k8sclientSet.CoreV1().Secrets("test")
	err := createResources(
		stack,
		&stackV1Beta2{stacks: &fakeV1beta2Client{errorOnCreate: true}},
		configs,
		secrets)
	assert.Error(t, err, "some error")
	_, err = configs.Get("test", metav1.GetOptions{})
	assert.Check(t, kerrors.IsNotFound(err))
	_, err = secrets.Get("test", metav1.GetOptions{})
	assert.Check(t, kerrors.IsNotFound(err))
}

type fakeV1beta1Client struct {
	errorOnCreate bool
}

func (c *fakeV1beta1Client) Create(s *v1beta1.Stack) (*v1beta1.Stack, error) {
	if c.errorOnCreate {
		return nil, errors.New("some error")
	}
	return s, nil
}

func (c *fakeV1beta1Client) Update(*v1beta1.Stack) (*v1beta1.Stack, error) {
	return nil, nil
}

func (c *fakeV1beta1Client) UpdateStatus(*v1beta1.Stack) (*v1beta1.Stack, error) {
	return nil, nil
}

func (c *fakeV1beta1Client) Delete(name string, options *metav1.DeleteOptions) error {
	return nil
}

func (c *fakeV1beta1Client) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return nil
}

func (c *fakeV1beta1Client) Get(name string, options metav1.GetOptions) (*v1beta1.Stack, error) {
	return nil, kerrors.NewNotFound(v1beta1.SchemeGroupVersion.WithResource("stacks").GroupResource(), name)
}

func (c *fakeV1beta1Client) List(opts metav1.ListOptions) (*v1beta1.StackList, error) {
	return nil, nil
}

func (c *fakeV1beta1Client) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}

func (c *fakeV1beta1Client) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*v1beta1.Stack, error) {
	return nil, nil
}

func (c *fakeV1beta1Client) WithSkipValidation() composev1beta1.StackInterface {
	return c
}

type fakeV1beta2Client struct {
	errorOnCreate bool
}

func (c *fakeV1beta2Client) Create(s *v1beta2.Stack) (*v1beta2.Stack, error) {
	if c.errorOnCreate {
		return nil, errors.New("some error")
	}
	return s, nil
}

func (c *fakeV1beta2Client) Update(*v1beta2.Stack) (*v1beta2.Stack, error) {
	return nil, nil
}

func (c *fakeV1beta2Client) UpdateStatus(*v1beta2.Stack) (*v1beta2.Stack, error) {
	return nil, nil
}

func (c *fakeV1beta2Client) Delete(name string, options *metav1.DeleteOptions) error {
	return nil
}

func (c *fakeV1beta2Client) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return nil
}

func (c *fakeV1beta2Client) Get(name string, options metav1.GetOptions) (*v1beta2.Stack, error) {
	return nil, kerrors.NewNotFound(v1beta1.SchemeGroupVersion.WithResource("stacks").GroupResource(), name)
}

func (c *fakeV1beta2Client) List(opts metav1.ListOptions) (*v1beta2.StackList, error) {
	return nil, nil
}

func (c *fakeV1beta2Client) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}

func (c *fakeV1beta2Client) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*v1beta2.Stack, error) {
	return nil, nil
}

func (c *fakeV1beta2Client) WithSkipValidation() composev1beta2.StackInterface {
	return c
}
