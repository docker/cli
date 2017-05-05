package client

import (
	"io"
	"net/url"

	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
)

// PluginLoad loads a plugin
func (cli *Client) PluginLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
	v := url.Values{}
	v.Set("quiet", "0")
	if quiet {
		v.Set("quiet", "1")
	}

	// set the type of the data request
	headers := map[string][]string{"Content-Type": {"application/x-tar"}}

	resp, err := cli.postRaw(ctx, "/plugins/load", v, input, headers)
	if err != nil {
		return types.ImageLoadResponse{}, err
	}

	return types.ImageLoadResponse{
		Body: resp.body,
		JSON: resp.header.Get("Content-Type") == "application/json",
	}, nil
}
