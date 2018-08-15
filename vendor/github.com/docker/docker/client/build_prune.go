package client // import "github.com/docker/docker/client"

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/docker/docker/api/types"
)

// BuildCachePrune requests the daemon to delete unused cache data
func (cli *Client) BuildCachePrune(ctx context.Context, opts types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error) {
	if err := cli.NewVersionError("1.31", "build prune"); err != nil {
		return nil, err
	}

	report := types.BuildCachePruneReport{}

	query := url.Values{}
	if opts.All {
		query.Set("all", "1")
	}
	query.Set("keep-duration", fmt.Sprintf("%d", opts.KeepDuration))
	query.Set("keep-storage", fmt.Sprintf("%f", opts.KeepStorage))

	serverResp, err := cli.post(ctx, "/build/prune", query, nil, nil)
	if err != nil {
		return nil, err
	}
	defer ensureReaderClosed(serverResp)

	if err := json.NewDecoder(serverResp.body).Decode(&report); err != nil {
		return nil, fmt.Errorf("Error retrieving disk usage: %v", err)
	}

	return &report, nil
}
