package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetAPIVersion(t *testing.T) {
	var tests = []struct {
		description   string
		groups        *metav1.APIGroupList
		err           bool
		expectedStack StackVersion
	}{
		{"no stack api", makeGroups(), true, ""},
		{"v1beta1", makeGroups(groupVersion{"compose.docker.com", []string{"v1beta1"}}), false, StackAPIV1Beta1},
		{"v1beta2", makeGroups(groupVersion{"compose.docker.com", []string{"v1beta2"}}), false, StackAPIV1Beta2},
		{"most recent has precedence", makeGroups(groupVersion{"compose.docker.com", []string{"v1beta1", "v1beta2"}}), false, StackAPIV1Beta2},
	}

	for _, test := range tests {
		version, err := getAPIVersion(test.groups)
		if test.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		assert.Equal(t, test.expectedStack, version)
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
