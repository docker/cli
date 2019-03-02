package kubernetes

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/compose-on-kubernetes/api/compose/v1alpha3"
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta1"
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewStackConverter(t *testing.T) {
	_, err := NewStackConverter("v1alpha1")
	assert.Check(t, is.ErrorContains(err, "stack version v1alpha1 unsupported"))

	_, err = NewStackConverter("v1beta1")
	assert.NilError(t, err)
	_, err = NewStackConverter("v1beta2")
	assert.NilError(t, err)
	_, err = NewStackConverter("v1alpha3")
	assert.NilError(t, err)
}

func TestConvertFromToV1beta1(t *testing.T) {
	composefile := `version: "3.3"
services: 
  test:
    image: nginx
secrets:
  test:
    file: testdata/secret
configs:
  test:
    file: testdata/config
`
	stackv1beta1 := &v1beta1.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1beta1.StackSpec{
			ComposeFile: composefile,
		},
	}

	result, err := stackFromV1beta1(stackv1beta1)
	assert.NilError(t, err)
	expected := Stack{
		Name:        "test",
		ComposeFile: composefile,
		Spec: &v1alpha3.StackSpec{
			Services: []v1alpha3.ServiceConfig{
				{
					Name:        "test",
					Image:       "nginx",
					Environment: make(map[string]*string),
				},
			},
			Secrets: map[string]v1alpha3.SecretConfig{
				"test": {File: filepath.FromSlash("testdata/secret")},
			},
			Configs: map[string]v1alpha3.ConfigObjConfig{
				"test": {File: filepath.FromSlash("testdata/config")},
			},
		},
	}
	assert.DeepEqual(t, expected, result)
	assert.DeepEqual(t, stackv1beta1, stackToV1beta1(result))
}

func TestConvertFromToV1beta2(t *testing.T) {
	stackv1beta2 := &v1beta2.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: &v1beta2.StackSpec{
			Services: []v1beta2.ServiceConfig{
				{
					Name:        "test",
					Image:       "nginx",
					Environment: make(map[string]*string),
				},
			},
			Secrets: map[string]v1beta2.SecretConfig{
				"test": {File: filepath.FromSlash("testdata/secret")},
			},
			Configs: map[string]v1beta2.ConfigObjConfig{
				"test": {File: filepath.FromSlash("testdata/config")},
			},
		},
	}
	expected := Stack{
		Name: "test",
		Spec: &v1alpha3.StackSpec{
			Services: []v1alpha3.ServiceConfig{
				{
					Name:        "test",
					Image:       "nginx",
					Environment: make(map[string]*string),
				},
			},
			Secrets: map[string]v1alpha3.SecretConfig{
				"test": {File: filepath.FromSlash("testdata/secret")},
			},
			Configs: map[string]v1alpha3.ConfigObjConfig{
				"test": {File: filepath.FromSlash("testdata/config")},
			},
		},
	}
	result, err := stackFromV1beta2(stackv1beta2)
	assert.NilError(t, err)
	assert.DeepEqual(t, expected, result)
	gotBack, err := stackToV1beta2(result)
	assert.NilError(t, err)
	assert.DeepEqual(t, stackv1beta2, gotBack)
}

func TestConvertFromToV1alpha3(t *testing.T) {
	stackv1alpha3 := &v1alpha3.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: &v1alpha3.StackSpec{
			Services: []v1alpha3.ServiceConfig{
				{
					Name:        "test",
					Image:       "nginx",
					Environment: make(map[string]*string),
				},
			},
			Secrets: map[string]v1alpha3.SecretConfig{
				"test": {File: filepath.FromSlash("testdata/secret")},
			},
			Configs: map[string]v1alpha3.ConfigObjConfig{
				"test": {File: filepath.FromSlash("testdata/config")},
			},
		},
	}
	expected := Stack{
		Name: "test",
		Spec: &v1alpha3.StackSpec{
			Services: []v1alpha3.ServiceConfig{
				{
					Name:        "test",
					Image:       "nginx",
					Environment: make(map[string]*string),
				},
			},
			Secrets: map[string]v1alpha3.SecretConfig{
				"test": {File: filepath.FromSlash("testdata/secret")},
			},
			Configs: map[string]v1alpha3.ConfigObjConfig{
				"test": {File: filepath.FromSlash("testdata/config")},
			},
		},
	}
	result := stackFromV1alpha3(stackv1alpha3)
	assert.DeepEqual(t, expected, result)
	gotBack := stackToV1alpha3(result)
	assert.DeepEqual(t, stackv1alpha3, gotBack)
}

func loadTestStackWith(t *testing.T, with string) *composetypes.Config {
	t.Helper()
	filePath := fmt.Sprintf("testdata/compose-with-%s.yml", with)
	data, err := ioutil.ReadFile(filePath)
	assert.NilError(t, err)
	yamlData, err := loader.ParseYAML(data)
	assert.NilError(t, err)
	cfg, err := loader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{Config: yamlData, Filename: filePath},
		},
	})
	assert.NilError(t, err)
	return cfg
}

func TestHandlePullSecret(t *testing.T) {
	testData := loadTestStackWith(t, "pull-secret")
	cases := []struct {
		version string
		err     string
	}{
		{version: "v1beta1", err: `stack API version v1beta1 does not support pull secrets (field "x-pull-secret"), please use version v1alpha3 or higher`},
		{version: "v1beta2", err: `stack API version v1beta2 does not support pull secrets (field "x-pull-secret"), please use version v1alpha3 or higher`},
		{version: "v1alpha3"},
	}

	for _, c := range cases {
		t.Run(c.version, func(t *testing.T) {
			conv, err := NewStackConverter(c.version)
			assert.NilError(t, err)
			s, err := conv.FromCompose(ioutil.Discard, "test", testData)
			if c.err != "" {
				assert.Error(t, err, c.err)

			} else {
				assert.NilError(t, err)
				assert.Equal(t, s.Spec.Services[0].PullSecret, "some-secret")
			}
		})
	}
}

func TestHandlePullPolicy(t *testing.T) {
	testData := loadTestStackWith(t, "pull-policy")
	cases := []struct {
		version string
		err     string
	}{
		{version: "v1beta1", err: `stack API version v1beta1 does not support pull policies (field "x-pull-policy"), please use version v1alpha3 or higher`},
		{version: "v1beta2", err: `stack API version v1beta2 does not support pull policies (field "x-pull-policy"), please use version v1alpha3 or higher`},
		{version: "v1alpha3"},
	}

	for _, c := range cases {
		t.Run(c.version, func(t *testing.T) {
			conv, err := NewStackConverter(c.version)
			assert.NilError(t, err)
			s, err := conv.FromCompose(ioutil.Discard, "test", testData)
			if c.err != "" {
				assert.Error(t, err, c.err)

			} else {
				assert.NilError(t, err)
				assert.Equal(t, s.Spec.Services[0].PullPolicy, "Never")
			}
		})
	}
}
