package licensing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/docker/licensing/lib/errors"
	"github.com/docker/licensing/lib/go-clientlib"
	"github.com/docker/licensing/model"
)

func (c *client) getUserByName(ctx context.Context, username string) (*model.User, error) {
	url := c.baseURI
	url.Path += fmt.Sprintf("/v2/users/%s/", username)
	response := new(model.User)
	_, _, err := c.doRequestNoAuth(ctx, "GET", &url, clientlib.RecvJSON(response))
	return response, err
}

func (c *client) getUserOrgs(ctx context.Context, params model.PaginationParams) ([]model.Org, error) {
	values := url.Values{}

	if params.PageSize != 0 {
		values.Set("page_size",
			fmt.Sprintf("%v", params.PageSize))
	}

	if params.Page != 0 {
		values.Set("page",
			fmt.Sprintf("%v", params.Page))
	}

	requrl := c.baseURI
	requrl.Path = "/v2/user/orgs/"
	requrl.RawQuery = values.Encode()

	var response struct {
		model.PaginatedMeta
		Results []model.Org `json:"results"`
	}
	_, _, err := c.doReq(ctx, "GET", &requrl, clientlib.RecvJSON(&response))
	return response.Results, err
}

// login calls the login endpoint
// If an error is returned by the Accounts service (as opposed to connection error, json unmarshalling error etc.),
// `error` will be of type `LoginError`
func (c *client) login(ctx context.Context, username string, password string) (*model.LoginResult, error) {
	url := c.baseURI
	url.Path += "/v2/users/login/"
	request := model.LoginRequest{
		Username: username,
		Password: password,
	}
	response := new(model.LoginResult)
	_, _, err := c.doRequestNoAuth(ctx, "POST", &url, clientlib.SendJSON(request), clientlib.RecvJSON(response), loginErrorCheckOpt)
	return response, err
}

// loginErrorCheckOpt works similarly to `clientlib.DefaultErrorCheck`, except it parses the error response
func loginErrorCheckOpt(r *clientlib.Request) {
	r.ErrorCheck = func(r *clientlib.Request, doErr error, res *http.Response) error {
		if doErr != nil {
			return errors.Wrap(doErr, r.ErrorFields())
		}
		status := res.StatusCode
		if status >= 200 && status < 300 {
			return nil
		}

		defer res.Body.Close()

		lError := new(model.LoginError)

		message := fmt.Sprintf("%s %s returned %d", r.Method, r.URL.String(), status)
		lError.HTTPError = errors.NewHTTPError(status, message).
			With(r.ErrorFields())

		var rawLoginErr model.RawLoginError
		err := json.NewDecoder(res.Body).Decode(&rawLoginErr)
		if err == nil {
			lError.Raw = &rawLoginErr
		}

		return lError
	}
}
