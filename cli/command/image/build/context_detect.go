package build

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli/command/image/build/internal/urlutil"
)

// ContextType describes the type (source) of build-context specified.
type ContextType string

const (
	ContextTypeStdin  ContextType = "stdin"  // ContextTypeStdin indicates that the build-context is a TAR archive passed through STDIN.
	ContextTypeLocal  ContextType = "local"  // ContextTypeLocal indicates that the build-context is a local directory.
	ContextTypeRemote ContextType = "remote" // ContextTypeRemote indicates that the build-context is a remote URL.
	ContextTypeGit    ContextType = "git"    // ContextTypeGit indicates that the build-context is a GIT URL.
)

// DetectContextType detects the type (source) of the build-context.
func DetectContextType(specifiedContext string) (ContextType, error) {
	switch {
	case specifiedContext == "-":
		return ContextTypeStdin, nil
	case isLocalDir(specifiedContext):
		return ContextTypeLocal, nil
	case urlutil.IsGitURL(specifiedContext):
		return ContextTypeGit, nil
	case urlutil.IsURL(specifiedContext):
		return ContextTypeRemote, nil
	default:
		return "", fmt.Errorf("unable to prepare context: path %q not found", specifiedContext)
	}
}

func isLocalDir(c string) bool {
	_, err := os.Stat(c)
	return err == nil
}
