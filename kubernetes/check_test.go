package kubernetes

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetStackAPIVersion(t *testing.T) {
	var tests = []struct {
		description   string
		groups        *metav1.APIGroupList
		experimental  bool
		err           bool
		expectedStack StackVersion
	}{
		{"no stack api", makeGroups(), false, true, ""},
		{"v1beta1", makeGroups(groupVersion{"compose.docker.com", []string{"v1beta1"}}), false, false, StackAPIV1Beta1},
		{"v1beta2", makeGroups(groupVersion{"compose.docker.com", []string{"v1beta2"}}), false, false, StackAPIV1Beta2},
		{"most recent has precedence", makeGroups(groupVersion{"compose.docker.com", []string{"v1beta1", "v1beta2"}}), false, false, StackAPIV1Beta2},
		{"most recent has precedence", makeGroups(groupVersion{"compose.docker.com", []string{"v1beta1", "v1beta2", "v1alpha3"}}), false, false, StackAPIV1Beta2},
		{"most recent has precedence", makeGroups(groupVersion{"compose.docker.com", []string{"v1beta1", "v1beta2", "v1alpha3"}}), true, false, StackAPIV1Alpha3},
	}

	for _, test := range tests {
		version, err := getAPIVersion(test.groups, test.experimental)
		if test.err {
			assert.ErrorContains(t, err, "")
		} else {
			assert.NilError(t, err)
		}
		assert.Check(t, is.Equal(test.expectedStack, version))
	}
}

type groupVersion struct {
	name     string
	versions []string
}

func makeGroups(versions ...groupVersion) *metav1.APIGroupList {
	groups := make([]metav1.APIGroup, len(versions))
	for i := range versions {
		groups[i].Name = versions[i].name
		for _, v := range versions[i].versions {
			groups[i].Versions = append(groups[i].Versions, metav1.GroupVersionForDiscovery{Version: v})
		}
	}
	return &metav1.APIGroupList{
		Groups: groups,
	}
}
