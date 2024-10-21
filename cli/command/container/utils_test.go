package container

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types/container"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func waitFn(cid string) (<-chan container.WaitResponse, <-chan error) {
	resC := make(chan container.WaitResponse)
	errC := make(chan error, 1)
	var res container.WaitResponse

	go func() {
		switch {
		case strings.Contains(cid, "exit-code-42"):
			res.StatusCode = 42
			resC <- res
		case strings.Contains(cid, "non-existent"):
			err := fmt.Errorf("no such container: %v", cid)
			errC <- err
		case strings.Contains(cid, "wait-error"):
			res.Error = &container.WaitExitError{Message: "removal failed"}
			resC <- res
		default:
			// normal exit
			resC <- res
		}
	}()

	return resC, errC
}

func TestWaitExitOrRemoved(t *testing.T) {
	tests := []struct {
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

	client := &fakeClient{waitFunc: waitFn, Version: api.DefaultVersion}
	for _, tc := range tests {
		t.Run(tc.cid, func(t *testing.T) {
			statusC := waitExitOrRemoved(context.Background(), client, tc.cid, true)
			exitCode := <-statusC
			assert.Check(t, is.Equal(tc.exitCode, exitCode))
		})
	}
}
