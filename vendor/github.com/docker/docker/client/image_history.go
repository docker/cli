package client // import "github.com/docker/docker/client"

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/docker/docker/api/types/image"
	"github.com/pkg/errors"
)

// ImageHistory returns the changes in an image in history format.
//
// Note: In future versions the opts parameter will be a single, non-variadic parameter.
// TODO: Change to ImageHistory(ctx context.Context, imageID string, opts images.HistoryOptions) ([]image.HistoryResponseItem, error)
func (cli *Client) ImageHistory(ctx context.Context, imageID string, opts ...image.HistoryOptions) ([]image.HistoryResponseItem, error) {
	if len(opts) > 1 {
		return nil, errors.New("only one HistoryOptions is supported")
	}

	values := url.Values{}
	if len(opts) == 1 && opts[0].Platform != nil {
		if err := cli.NewVersionError(ctx, "1.47", "platform"); err != nil {
			return nil, err
		}

		p, err := json.Marshal(*opts[0].Platform)
		if err != nil {
			return nil, fmt.Errorf("invalid platform: %v", err)
		}
		values.Set("platform", string(p))
	}

	var history []image.HistoryResponseItem
	serverResp, err := cli.get(ctx, "/images/"+imageID+"/history", values, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return history, err
	}

	err = json.NewDecoder(serverResp.body).Decode(&history)
	return history, err
}
