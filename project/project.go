package project

import (
	"io/ioutil"
	"net"
	"net/http"

	"os"

	"github.com/spf13/pflag"
)

var (
	// CurrentProject is used to store loaded project
	currentProject Project
)

// Project is the interface used by the cli package
type Project interface {
	RootDir() string         // returns the project root directory's path
	ID() string              // returns project id
	Dial() (net.Conn, error) // returns conn to proxy
	NewScopedHttpClient(backendAddr, apiVersion string) (*http.Client, error)
}

// IsInProject indicates whether we are in the context of a project
func IsInProject() bool {
	return GetCurrentProject() != nil
}

func GetCurrentProject() Project {
	if isProjectIgnored() {
		return nil
	}
	return currentProject
}

func SetCurrentProject(p Project) {
	currentProject = p
}

func isProjectIgnored() bool {
	var ignoreProjectFlag bool = false
	f := pflag.NewFlagSet("", pflag.ContinueOnError)
	f.SetOutput(ioutil.Discard)
	f.BoolVar(&ignoreProjectFlag, "ignore-project", false, "disables project scoping")
	_ = f.Parse(os.Args[1:])
	return ignoreProjectFlag
}
