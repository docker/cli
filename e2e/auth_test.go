package e2e

import (
	"testing"

	"github.com/docker/cli/e2e/internal/registry"
	"github.com/docker/cli/internal/test/command"
)

func runDockerCommand(t *testing.T, args ...string) {
	cmd := command.NewDockerCommand(t, args...)
	cmd.Assert()
}

func TestAuthorizedPullPush(t *testing.T) {
	const (
		username = "testuser"
		password = "testpassword"
	)
	reg, err := registry.NewV2(
		registry.WithAuth(username, password),
	)
	if err != nil {
		t.Fatalf("Failed to start registry: %v", err)
	}
	defer reg.Stop()

	repo := reg.RepoName("private/alpine")

	// docker login
	runDockerCommand(t, "login", reg.Host(), "-u", username, "-p", password)
	runDockerCommand(t, "pull", "alpine")
	runDockerCommand(t, "tag", "alpine", repo)
	runDockerCommand(t, "push", repo)
	runDockerCommand(t, "rmi", repo)
	runDockerCommand(t, "pull", repo)
}
