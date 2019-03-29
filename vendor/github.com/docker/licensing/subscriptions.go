package licensing

import (
	"context"
	"net/url"

	"github.com/docker/licensing/lib/go-clientlib"
	"github.com/docker/licensing/model"
)

// RequestParams holds request parameters
type RequestParams struct {
	DockerID         string
	PartnerAccountID string
	Origin           string
}

func (c *client) createSubscription(ctx context.Context, request *model.SubscriptionCreationRequest) (*model.SubscriptionDetail, error) {
	url := c.baseURI
	url.Path += "/api/billing/v4/subscriptions"
	response := new(model.SubscriptionDetail)
	if _, _, err := c.doReq(ctx, "POST", &url, clientlib.SendJSON(request), clientlib.RecvJSON(response)); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *client) getSubscription(ctx context.Context, id string) (*model.SubscriptionDetail, error) {
	url := c.baseURI
	url.Path += "/api/billing/v4/subscriptions/" + id
	response := new(model.SubscriptionDetail)
	if _, _, err := c.doReq(ctx, "GET", &url, clientlib.RecvJSON(response)); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *client) listSubscriptions(ctx context.Context, params map[string]string) ([]*model.Subscription, error) {
	values := url.Values{}
	values.Set("docker_id", params["docker_id"])
	values.Set("partner_account_id", params["partner_account_id"])
	values.Set("origin", params["origin"])

	url := c.baseURI
	url.Path += "/api/billing/v4/subscriptions"
	url.RawQuery = values.Encode()

	response := make([]*model.Subscription, 0)
	if _, _, err := c.doReq(ctx, "GET", &url, clientlib.RecvJSON(&response)); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *client) listSubscriptionsDetails(ctx context.Context, params map[string]string) ([]*model.SubscriptionDetail, error) {
	values := url.Values{}
	values.Set("docker_id", params["docker_id"])
	values.Set("partner_account_id", params["partner_account_id"])
	values.Set("origin", params["origin"])

	url := c.baseURI
	url.Path += "/api/billing/v4/subscriptions"
	url.RawQuery = values.Encode()

	response := make([]*model.SubscriptionDetail, 0)
	if _, _, err := c.doReq(ctx, "GET", &url, clientlib.RecvJSON(&response)); err != nil {
		return nil, err
	}

	return response, nil
}
