package client // import "github.com/docker/docker/client"

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/docker/docker/api/types/image"
)

// ImageSave retrieves one or more images from the docker host as an io.ReadCloser.
// It's up to the caller to store the images and close the stream.
// Note: In future versions the opts parameter will be a single, non-variadic parameter.
// TODO: Change to ImageSave(ctx context.Context, imageIDs []string, opts image.SaveOptions) (io.ReadCloser, error) {
func (cli *Client) ImageSave(ctx context.Context, imageIDs []string, opts ...image.SaveOptions) (io.ReadCloser, error) {
	query := url.Values{
		"names": imageIDs,
	}

	if len(opts) == 1 && opts[0].Platform != nil {
		if err := cli.NewVersionError(ctx, "1.47", "platform"); err != nil {
			return nil, err
		}

		p, err := json.Marshal(*opts[0].Platform)
		if err != nil {
			return nil, fmt.Errorf("invalid platform: %v", err)
		}
		query.Set("platform", string(p))
	}

	resp, err := cli.get(ctx, "/images/get", query, nil)
	if err != nil {
		return nil, err
	}
	return resp.body, nil
}
