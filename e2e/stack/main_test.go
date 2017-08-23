package stack

import (
	"fmt"
	"os"
	"testing"

	"github.com/pkg/errors"
)

func TestMain(m *testing.M) {
	if err := setupTestEnv(); err != nil {
		fmt.Println(err.Error())
		os.Exit(3)
	}
	os.Exit(m.Run())
}

// TODO: move to shared internal package
func setupTestEnv() error {
	dockerHost := os.Getenv("TEST_DOCKER_HOST")
	if dockerHost == "" {
		return errors.New("$TEST_DOCKER_HOST must be set")
	}
	return os.Setenv("DOCKER_HOST", dockerHost)
}
