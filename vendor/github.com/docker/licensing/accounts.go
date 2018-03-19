package licensing

import (
	"context"

	"github.com/docker/licensing/lib/go-clientlib"
	"github.com/docker/licensing/model"
)

func (c *client) createAccount(ctx context.Context, dockerID string, request *model.AccountCreationRequest) (*model.Account, error) {
	url := c.baseURI
	url.Path += "/api/billing/v4/accounts/" + dockerID

	response := new(model.Account)
	if _, _, err := c.doReq(ctx, "PUT", &url, clientlib.SendJSON(request), clientlib.RecvJSON(response)); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *client) getAccount(ctx context.Context, dockerID string) (*model.Account, error) {
	url := c.baseURI
	url.Path += "/api/billing/v4/accounts/" + dockerID

	response := new(model.Account)
	if _, _, err := c.doReq(ctx, "GET", &url, clientlib.RecvJSON(response)); err != nil {
		return nil, err
	}

	return response, nil
}
