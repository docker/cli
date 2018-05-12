package container

import (
	"context"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types/container"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func waitFn(cid string) (<-chan container.ContainerWaitOKBody, <-chan error) {
	resC := make(chan container.ContainerWaitOKBody)
	errC := make(chan error, 1)
	var res container.ContainerWaitOKBody

	go func() {
		switch {
		case strings.Contains(cid, "exit-code-42"):
			res.StatusCode = 42
			resC <- res
		case strings.Contains(cid, "non-existent"):
			err := errors.Errorf("No such container: %v", cid)
			errC <- err
		case strings.Contains(cid, "wait-error"):
			res.Error = &container.ContainerWaitOKBodyError{Message: "removal failed"}
			resC <- res
		default:
			// normal exit
			resC <- res
		}
	}()

	return resC, errC
}

func TestWaitExitOrRemoved(t *testing.T) {
	testcases := []struct {
		cid      string
		exitCode int
	}{
		{
			cid:      "normal-container",
			exitCode: 0,
		},
		{
			cid:      "give-me-exit-code-42",
			exitCode: 42,
		},
		{
			cid:      "i-want-a-wait-error",
			exitCode: 125,
		},
		{
			cid:      "non-existent-container-id",
			exitCode: 125,
		},
	}

	client := test.NewFakeCli(&fakeClient{waitFunc: waitFn, Version: api.DefaultVersion})
	for _, testcase := range testcases {
		statusC := waitExitOrRemoved(context.Background(), client, testcase.cid, true)
		exitCode := <-statusC
		assert.Check(t, is.Equal(testcase.exitCode, exitCode))
	}
}
