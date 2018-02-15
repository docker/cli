package util

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/project"
)

const (
	recentProjectsFileName = ".recentProjects.json"
)

// ChRootDir changes the current working directory to project's root directory
func ChRootDir(p project.Project) (previousWorkDir string, err error) {
	previousWorkDir, err = os.Getwd()
	if err != nil {
		return
	}
	err = os.Chdir(p.RootDir())
	if err != nil {
		return
	}
	return
}

type recentProject struct {
	IDVal      string `json:"id"`
	RootDirVal string `json:"root"`
	Timestamp  int    `json:"t"`
}

// Project interface implementation
func (rp *recentProject) RootDir() string {
	return rp.RootDirVal
}
func (rp *recentProject) ID() string {
	return rp.IDVal
}
func (rp *recentProject) Dial() (net.Conn, error) {
	return nil, nil
}
func (rp *recentProject) NewScopedHTTPClient(backendAddr, apiVersion string) (*http.Client, error) {
	return nil, nil
}

type recentProjects []*recentProject

func (a recentProjects) Len() int           { return len(a) }
func (a recentProjects) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a recentProjects) Less(i, j int) bool { return a[i].Timestamp > a[j].Timestamp }

// SaveInRecentProjects inserts project or updates existing entry in the
// file storing recent projects
func SaveInRecentProjects(p project.Project) error {
	rProjects := getRecentProjects()
	inserted := false

	rp := &recentProject{
		IDVal:      p.ID(),
		RootDirVal: p.RootDir(),
		Timestamp:  int(time.Now().Unix()),
	}

	for i, rProject := range rProjects {
		if rProject.IDVal == rp.IDVal {
			rProjects[i] = rp
			inserted = true
			break
		}
	}

	if !inserted {
		rProjects = append(rProjects, rp)
	}

	err := saveRecentProjects(rProjects)
	if err != nil {
		return err
	}

	return nil
}

// RemoveFromRecentProjects removes project at given root dir path from
// recent project. Using a string parameter instead of project.Project
// because the actual directory may have been removed manually while the
// user still wants to remove the project from the list...
func RemoveFromRecentProjects(rootDir string) error {
	rProjects := getRecentProjects()
	removed := false

	for i, rProject := range rProjects {
		if rProject.RootDir() == rootDir {
			// last element
			if i == len(rProjects)-1 {
				rProjects = rProjects[:i]
			} else {
				rProjects = append(rProjects[:i], rProjects[i+1:]...)
			}
			removed = true
			break
		}
	}

	if !removed {
		return errors.New("can't remove from recent project (not found)")
	}

	err := saveRecentProjects(rProjects)
	if err != nil {
		return err
	}

	return nil
}

// GetRecentProjects returns the ordered list of recent projects.
func GetRecentProjects() []project.Project {
	rp := getRecentProjects()
	resp := make([]project.Project, len(rp))
	for i, p := range rp {
		resp[i] = p
	}
	return resp
}

func getRecentProjects() recentProjects {
	rProjects := make(recentProjects, 0)
	jsonBytes, err := ioutil.ReadFile(recentProjectsFilePath())
	if err == nil {
		err := json.Unmarshal(jsonBytes, &rProjects)
		if err != nil {
			return make(recentProjects, 0)
		}
	}
	return rProjects
}

func recentProjectsFilePath() string {
	return filepath.Join(cliconfig.Dir(), recentProjectsFileName)
}

func saveRecentProjects(rProjects recentProjects) error {
	// filter to enforce only one project per location
	locationFilter := make(map[string]*recentProject)

	for _, rProject := range rProjects {
		if p, exists := locationFilter[rProject.RootDir()]; exists {
			// keep most recent
			if p.Timestamp > rProject.Timestamp {
				continue
			}
		}
		locationFilter[rProject.RootDir()] = rProject
	}

	filteredRecentProjects := make(recentProjects, len(locationFilter))

	i := 0
	for _, filteredProject := range locationFilter {
		filteredRecentProjects[i] = filteredProject
		i++
	}

	sort.Sort(filteredRecentProjects)

	jsonBytes, err := json.Marshal(filteredRecentProjects)
	if err != nil {
		return err
	}
	// update recent projects JSON file on disc
	// automatically create parent directories if they don't exist
	filePath := recentProjectsFilePath()
	parentDirPath := filepath.Dir(filePath)
	err = os.MkdirAll(parentDirPath, os.ModePerm)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, jsonBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
