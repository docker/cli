package container

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
)

func TestContainerPrunePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		containerPruneFunc: func(ctx context.Context, opts client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
			return client.ContainerPruneResult{}, errors.New("fakeClient containerPruneFunc should not be called")
		},
	})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
}

// pruneReportReader checks that progress was printed before the final report is read.
type pruneRoundTripper func(*http.Request) (*http.Response, error)

func (r pruneRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return r(req) }

type pruneReportReader func([]byte) (int, error)

func (r pruneReportReader) Read(p []byte) (int, error) { return r(p) }
func TestContainerPruneProgressBeforeReport(t *testing.T) {
	var cli *test.FakeCli
	final := strings.NewReader(`{"report":{"ContainersDeleted":["removed"],"SpaceReclaimed":42}}`)
	realClient, err := client.New(client.WithAPIVersion("1.56"), client.WithHTTPClient(&http.Client{Transport: pruneRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/jsonl"}}, Body: io.NopCloser(io.MultiReader(
			strings.NewReader("{\"object\":{\"id\":\"removed\",\"action\":\"deleted\"}}\n"),
			pruneReportReader(func(p []byte) (int, error) {
				assert.Equal(t, cli.OutBuffer().String(), "Deleted Containers:\nremoved\n")
				return final.Read(p)
			}),
		))}, nil
	})}))
	assert.NilError(t, err)
	cli = test.NewFakeCli(&fakeClient{containerPruneFunc: func(ctx context.Context, opts client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
		assert.Assert(t, opts.Stream)
		return realClient.ContainerPrune(ctx, opts)
	}})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{"--force"})
	assert.NilError(t, cmd.Execute())
	assert.Equal(t, cli.OutBuffer().String(), "Deleted Containers:\nremoved\n\nTotal reclaimed space: 42B\n")
}
