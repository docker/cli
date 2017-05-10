package pipeline

import (
	"strings"

	"github.com/docker/docker/api/types/filters"
)

const (
	pipelineLabel = "com.docker.pipeline.scope"
	projectLabel  = "com.docker.project.id"
	// projectNameLabel = "com.docker.project.name"
)

// Namespace mangles names by prepending the name
type Namespace struct {
	label string
	name  string
}

// Scope prepends the namespace to a name
func (n Namespace) ScopeName(name string) string {
	if name == "" {
		return ""
	}
	return n.name + "_" + name
}

// Descope returns the name without the namespace prefix
func (n Namespace) DescopeName(name string) string {
	return strings.TrimPrefix(name, n.name+"_")
}

// Name returns the name of the namespace
func (n Namespace) Name() string {
	return n.name
}

func (n Namespace) IsInScope(labels map[string]string) bool {
	scope, ok := labels[n.label]
	return ok && n.name == scope
}

func (n Namespace) UpdateFilter(reqFilters filters.Args) {
	reqFilters.Add("label", n.label+"="+n.Name())
	for _, nameFilter := range reqFilters.Get("name") {
		reqFilters.Del("name", nameFilter)
		reqFilters.Add("name", n.ScopeName(nameFilter))
	}
}

// NewNamespace returns a new Namespace for scoping of names
func NewNamespace(name, label string) Namespace {
	return Namespace{name: name, label: label}
}

// LookupScope is a function which returns a ScopeInfo
type LookupScope func() Scoper

// Scoper provides scoping to middleware
type Scoper interface {
	ScopeName(string) string
	DescopeName(string) string
	AddLabels(map[string]string)
	UpdateFilter(filters.Args)
	IsInScope(map[string]string) bool
}

type pipelineScoper struct {
	Namespace
}

func (s *pipelineScoper) AddLabels(labels map[string]string) {
	labels[s.Namespace.label] = s.Namespace.Name()
}

func NewPipelineScoper(name string) Scoper {
	return &pipelineScoper{Namespace: NewNamespace(name, pipelineLabel)}
}

type projectScoper struct {
	Namespace
	humanName string
}

func (s *projectScoper) AddLabels(labels map[string]string) {
	labels[projectLabel] = s.Namespace.Name()
	// labels[projectNameLabel] = s.humanName
}

func NewProjectScoper(id string) Scoper {
	return &projectScoper{
		Namespace: NewNamespace(id, projectLabel),
	}
}
