package pipeline

import (
	gojson "encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/engine-api-proxy/json"
	"github.com/docker/engine-api-proxy/routes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func defaultLookup() Scoper {
	return NewProjectScoper("myproject")
}

type argsBuilder struct {
	args filters.Args
}

func newArgs() argsBuilder {
	return argsBuilder{args: filters.NewArgs()}
}

func (a argsBuilder) add(key, value string) argsBuilder {
	a.args.Add(key, value)
	return a
}

func (a argsBuilder) values(t *testing.T) url.Values {
	values := url.Values{}
	encoded, err := filters.ToParam(a.args)
	assert.NoError(t, err)
	values.Add("filters", encoded)
	return values
}

func TestObjectListRequest(t *testing.T) {
	expectedLabel := func() argsBuilder {
		return newArgs().add("label", projectLabel+"=myproject")
	}

	var cases = []struct {
		doc      string
		filters  argsBuilder
		expected argsBuilder
	}{
		{
			doc:      "No Filters",
			expected: expectedLabel(),
		},
		{
			doc:      "Filter by name",
			filters:  newArgs().add("name", "foo"),
			expected: expectedLabel().add("name", "myproject_foo"),
		},
		{
			doc: "Filter by labels and a name",
			filters: newArgs().add(
				"name", "foo").add(
				"label", "com.example=zoom"),
			expected: expectedLabel().add(
				"label", "com.example=zoom").add(
				"name", "myproject_foo"),
		},
	}

	for _, testcase := range cases {
		query := testcase.filters.values(t).Encode()
		input, err := http.NewRequest("GET", "/containers?"+query, nil)
		if !assert.NoError(t, err, testcase.doc) {
			continue
		}

		req, err := objectListRequest(defaultLookup, nil, input)
		if !assert.NoError(t, err, testcase.doc) {
			continue
		}
		if !assert.NoError(t, req.ParseForm(), testcase.doc) {
			continue
		}
		assert.Equal(t, testcase.expected.values(t), req.Form, testcase.doc)
	}
}

func TestObjectListResponse(t *testing.T) {
	volumes := []types.Volume{{Name: "myproject_foo", Scope: "local"}}
	expected := []types.Volume{{Name: "foo", Scope: "local"}}
	_, encoded, err := json.Encode(volumes)
	require.NoError(t, err)

	size, reader, err := objectListResponse(defaultLookup, nil, encoded)
	require.NoError(t, err)
	assert.Equal(t, 90, size)
	actual := []types.Volume{}
	err = gojson.NewDecoder(reader).Decode(&actual)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestObjectInspectResponse(t *testing.T) {
	volume := types.Volume{Name: "myproject_foo", Scope: "local"}
	expected := types.Volume{Name: "foo", Scope: "local"}
	_, encoded, err := json.Encode(volume)
	require.NoError(t, err)

	resp := &http.Response{StatusCode: http.StatusOK}
	size, reader, err := objectInspectResponse(defaultLookup, resp, encoded)
	require.NoError(t, err)
	assert.Equal(t, 88, size)
	actual := types.Volume{}
	err = gojson.NewDecoder(reader).Decode(&actual)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestObjectCreateRequest(t *testing.T) {
	var cases = []struct {
		doc          string
		name         string
		expectedName string
		labels       map[string]string
	}{
		{
			doc: "No name, no labels",
		},
		{
			doc:          "With a name",
			name:         "foo",
			expectedName: "myproject_foo",
		},
		{
			doc:    "With labels",
			labels: map[string]string{"something": "else"},
		},
	}

	for _, testcase := range cases {
		network := types.NetworkCreateRequest{
			Name:          testcase.name,
			NetworkCreate: types.NetworkCreate{Labels: testcase.labels},
		}
		labels := copyLabels(testcase.labels)
		labels[projectLabel] = "myproject"
		expected := types.NetworkCreateRequest{
			Name:          testcase.expectedName,
			NetworkCreate: types.NetworkCreate{Labels: labels},
		}

		_, body, err := json.Encode(network)
		if !assert.NoError(t, err) {
			continue
		}
		input, err := http.NewRequest("GET", "/network/create", body)
		if !assert.NoError(t, err, testcase.doc) {
			continue
		}
		req, err := objectCreateRequest(defaultLookup, nil, input)
		if !assert.NoError(t, err, testcase.doc) {
			continue
		}
		actual := types.NetworkCreateRequest{}
		err = gojson.NewDecoder(req.Body).Decode(&actual)
		if !assert.NoError(t, err, testcase.doc) {
			continue
		}
		assert.Equal(t, expected, actual)
	}
}

func copyLabels(source map[string]string) map[string]string {
	result := map[string]string{}
	if source == nil {
		return result
	}
	for k, v := range source {
		result[k] = v
	}
	return result
}

func TestScopePathRequest(t *testing.T) {
	scopeLabels := map[string]string{projectLabel: "myproject"}
	var cases = []struct {
		doc           string
		nameOrID      string
		labeled       *labeled
		verifyLabeled *labeled
		expected      string
		query         string
		expectError   string
	}{
		{
			doc:      "Scoped name should get double scoped",
			nameOrID: "myproject_gibs",
			labeled:  &labeled{ID: "aaaaaaa", Labels: scopeLabels},
			expected: "/containers/myproject_myproject_gibs",
		},
		{
			doc:      "ID should not be scoped",
			nameOrID: "aaaaaaa",
			labeled:  &labeled{ID: "aaaaaaa", Labels: scopeLabels},
			expected: "/containers/aaaaaaa",
		},
		{
			doc:      "Unscoped name should be scoped",
			nameOrID: "gibs",
			expected: "/containers/myproject_gibs?foo=zoom",
			query:    "?foo=zoom",
		},
		{
			doc:      "Unscoped name matches an unscoped object, needs to be scoped",
			nameOrID: "gibs",
			labeled:  &labeled{ID: "aaaaaaa", Labels: map[string]string{}},
			expected: "/containers/myproject_gibs",
		},
		{
			doc:      "Unscoped name is already in scope, should not be scoped",
			nameOrID: "random_name",
			labeled:  &labeled{ID: "aaaaaaa", Labels: scopeLabels},
			expected: "/containers/random_name?foo=zoom",
			query:    "?foo=zoom",
		},
		{
			doc:           "Scoped name matches an object that is out of scope",
			nameOrID:      "name",
			verifyLabeled: &labeled{ID: "aaaaaaa", Labels: map[string]string{}},
			expectError:   "\"name\" not found",
		},
	}

	for _, testcase := range cases {
		path := scopePath{
			lookup:    defaultLookup,
			inspector: newMockInspector(testcase.labeled, testcase.verifyLabeled),
			getVars: func(_ *http.Request) map[string]string {
				return map[string]string{
					"name":    testcase.nameOrID,
					"version": "v1.42",
				}
			},
		}
		route := routes.ContainerRemove.AsMuxRoute()
		urlStr := "/containers/" + testcase.nameOrID + testcase.query
		input, err := http.NewRequest("DELETE", urlStr, nil)
		if !assert.NoError(t, err, testcase.doc) {
			continue
		}

		req, err := path.request(route, input)
		if testcase.expectError != "" {
			if !assert.Error(t, err, testcase.doc) {
				continue
			}
			assert.Contains(t, err.Error(), testcase.expectError, testcase.doc)
			continue
		}

		if !assert.NoError(t, err, testcase.doc) {
			continue
		}
		assert.Equal(t, testcase.expected, req.URL.String(), testcase.doc)
	}
}

func newMockInspector(labeleds ...*labeled) inspector {
	index := 0
	return func(ctx context.Context, nameOrID string) (*labeled, error) {
		if index >= len(labeleds) {
			panic("MockInspector is out of labeled")
		}
		result := labeleds[index]
		index++
		if result == nil {
			return nil, notFound{}
		}
		return result, nil
	}
}

type notFound struct{}

func (n notFound) NotFound() bool { return true }
func (n notFound) Error() string  { return "oops" }
