package project

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	projectUtil "github.com/docker/cli/project/util"
	"github.com/docker/distribution/uuid"
	"github.com/docker/docker/client"
	proxyFakes "github.com/docker/engine-api-proxy/fakes"
	proxyNamespace "github.com/docker/engine-api-proxy/namespace"
	"github.com/docker/engine-api-proxy/proxy"
)

const (
	// ProjectDir describes where the project stores configuration
	ProjectDir = ".git/.docker"
	// ProjectIDFile is the name of the file storing project's id
	ProjectIDFile = "id"
)

// Project defines a Docker project.
// It implements the "github.com/docker/cli/proj".Project interface.
type Project struct {
	rootDir string
	proxy   *proxy.Proxy
}

// Init initiates a new project
func Init(dir, id string, force bool) (*Project, error) {
	if force == false && isProjectRoot(dir) {
		return nil, fmt.Errorf("target directory already is the root of a Docker project")
	}

	if id == "" {
		id = uuid.Generate().String()
	}

	if err := validateProjectID(id); err != nil {
		return nil, err
	}

	// write id file
	IDFile := filepath.Join(dir, ProjectDir, ProjectIDFile)
	err := os.MkdirAll(ProjectDir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(IDFile, []byte(id), 0644)
	if err != nil {
		return nil, err
	}

	proj, err := Load(dir)
	if err != nil {
		return nil, err
	}

	return proj, nil
}

// isProjectRoot looks for a project configuration file at a given path.
func isProjectRoot(dirPath string) (found bool) {
	found = false
	IDFile := filepath.Join(dirPath, ProjectDir, ProjectIDFile)
	fileInfo, err := os.Stat(IDFile)
	if os.IsNotExist(err) {
		return
	}
	if fileInfo.IsDir() {
		return
	}
	found = true
	return
}

// FindProjectRoot looks in current directory and parents until
// it finds a project config file. It then returns the parent
// of that directory, the root of the Docker project.
func FindProjectRoot(path string) (projectRootPath string, err error) {
	path = filepath.Clean(path)
	for {
		if isProjectRoot(path) {
			return path, nil
		}
		// break after / has been tested
		if path == filepath.Dir(path) {
			break
		}
		path = filepath.Dir(path)
	}
	return "", errors.New("can't find project root directory")
}

// RootDir returns project's root directory
func (p *Project) RootDir() string {
	return p.rootDir
}

// ID returns project's ID
func (p *Project) ID() string {
	id, err := p.getProjectID()
	if err != nil {
		log.Fatalln(err.Error())
	}
	return id
}

// Dial connects to the remote API through in-memory proxy to deal with
// labels and names.
func (p *Project) Dial() (net.Conn, error) {
	fakeListener, ok := p.proxy.GetListener().(*proxyFakes.FakeListener)
	if ok == false {
		return nil, errors.New("listener is not a fake listener")
	}
	return fakeListener.DialContext(nil, "", "")
}

// NewScopedHTTPClient creates a docker API client used internally by the proxy
func (p *Project) NewScopedHTTPClient(backendAddr, apiVersion string) (*http.Client, error) {
	//
	dockerClient, err := client.NewClient(backendAddr, apiVersion, nil, nil)
	if err != nil {
		return nil, err
	}
	// obtain proxy routes
	proxyRoutes := proxyNamespace.MiddlewareRoutes(dockerClient, func() proxyNamespace.Scoper {
		return proxyNamespace.NewProjectScoper(p.ID())
	})
	// construct proxy options struct
	proxyOpts := proxy.Options{
		Listen:      "",
		Backend:     backendAddr,
		SocketGroup: "",
		Routes:      proxyRoutes,
	}
	// create in-memory proxy
	p.proxy, err = proxy.NewInMemoryProxy(proxyOpts)
	if err != nil {
		return nil, err
	}
	// start proxy server goroutine
	go p.proxy.Start()

	// get FakeListener from proxy
	fakeListener, ok := p.proxy.GetListener().(*proxyFakes.FakeListener)
	if ok == false {
		return nil, errors.New("listener is not a fake listener")
	}

	transport := &http.Transport{}
	transport.DialContext = fakeListener.DialContext

	return &http.Client{
		Transport: transport,
	}, nil
}

// Leave deletes the file used to store project ID. That is a sufficient
// condition to stop working in the scope of a project.
func (p *Project) Leave() error {
	IDFilePath := filepath.Join(p.RootDir(), ProjectDir, ProjectIDFile)
	return os.Remove(IDFilePath)
}

// Load returns project for a given path.
// The configuration file can be in a parent directory, so we have to test all
// the way up to the root directory. If no configuration file is found then
// nil,nil is returned (no error)
func Load(path string) (*Project, error) {

	projectRootDirPath, err := FindProjectRoot(path)
	if err != nil {
		// TODO: gdevillele: handle actual errors, for now we suppose no project is found
		return nil, nil
	}

	// create project struct
	p := &Project{
		rootDir: projectRootDirPath,
	}

	// go to project root dir
	previousWorkDir, err := projectUtil.ChRootDir(p)
	if err != nil {
		return nil, err
	}
	defer os.Chdir(previousWorkDir)

	// validate project ID
	_, err = p.getProjectID()
	if err != nil {
		return nil, err
	}

	return p, nil
}

// LoadForWd returns project for current working directory
func LoadForWd() (*Project, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return Load(wd)
}

// getProjectID retrieves the ID of the project
// and validates it (it cannot contain any '.' for example)
func (p *Project) getProjectID() (string, error) {
	IDBytes, err := ioutil.ReadFile(filepath.Join(p.RootDir(), ProjectDir, ProjectIDFile))
	if err != nil {
		return "", errors.New("can't read project id")
	}
	ID := string(IDBytes)
	if err := validateProjectID(ID); err != nil {
		return "", err
	}
	return ID, nil
}

func validateProjectID(id string) error {
	rgxp := regexp.MustCompilePOSIX("^[0-9a-zA-Z_-]+$")
	if rgxp.MatchString(id) == false {
		return errors.New("invalid project id")
	}
	return nil
}
