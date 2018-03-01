package environment

import (
	"fmt"
	"os"
	"testing"
	"time"

	shlex "github.com/flynn-archive/go-shlex"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/gotestyourself/gotestyourself/poll"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
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

// Shell updates icmd.Command from the specified formatted string
// TODO: move to gotestyourself
func Shell(t *testing.T, format string, args ...interface{}) icmd.Cmd {
	cmd, err := shlex.Split(fmt.Sprintf(format, args...))
	require.NoError(t, err)
	return icmd.Cmd{Command: cmd}
}
