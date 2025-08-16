package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
)

// ReadDockerignore reads the .dockerignore file in the context directory and
// returns the list of paths to exclude
func ReadDockerignore(contextDir string) ([]string, error) {
	var excludes []string

	f, err := os.Open(filepath.Join(contextDir, ".dockerignore"))
	switch {
	case os.IsNotExist(err):
		return excludes, nil
	case err != nil:
		return nil, err
	}
	defer f.Close()

	patterns, err := ignorefile.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading .dockerignore: %w", err)
	}
	return patterns, nil
}

// TrimBuildFilesFromExcludes removes the named Dockerfile and .dockerignore from
// the list of excluded files. The daemon will remove them from the final context
// but they must be in available in the context when passed to the API.
func TrimBuildFilesFromExcludes(excludes []string, dockerfile string, dockerfileFromStdin bool) []string {
	if keep, _ := patternmatcher.Matches(".dockerignore", excludes); keep {
		excludes = append(excludes, "!.dockerignore")
	}

	// canonicalize dockerfile name to be platform-independent.
	dockerfile = filepath.ToSlash(dockerfile)
	if keep, _ := patternmatcher.Matches(dockerfile, excludes); keep && !dockerfileFromStdin {
		excludes = append(excludes, "!"+dockerfile)
	}
	return excludes
}
