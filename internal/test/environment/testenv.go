package environment

import (
	"os"
	"time"

	"github.com/gotestyourself/gotestyourself/poll"
	"github.com/pkg/errors"
)

// Setup a new environment
func Setup() error {
	dockerHost := os.Getenv("TEST_DOCKER_HOST")
	if dockerHost == "" {
		return errors.New("$TEST_DOCKER_HOST must be set")
	}
	return os.Setenv("DOCKER_HOST", dockerHost)
}

// DefaultPollSettings used with gotestyourself/poll
var DefaultPollSettings = poll.WithDelay(100 * time.Millisecond)
