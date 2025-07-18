package e2e

import (
	"os"
	"os/exec"
	"testing"
)

func runCmd(t *testing.T, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %s %v\nOutput: %s\nError: %v", name, args, output, err)
	}
	return string(output)
}

func TestAuthorizedPullPush(t *testing.T) {
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	privateRepo := os.Getenv("PRIVATE_REPO")

	if username == "" || password == "" || privateRepo == "" {
		t.Fatal("DOCKER_USERNAME, DOCKER_PASSWORD, and PRIVATE_REPO must be set")
	}

	runCmd(t, "docker", "login", "--username", username, "--password", password)
	runCmd(t, "docker", "pull", "alpine")
	runCmd(t, "docker", "tag", "alpine", privateRepo)
	runCmd(t, "docker", "push", privateRepo)
	runCmd(t, "docker", "rmi", privateRepo)
	runCmd(t, "docker", "pull", privateRepo)
}
