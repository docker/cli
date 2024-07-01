package util

import (
	"io"
	"net/http"
)

// Client is a client and actions for interacting with the tenant auth API.
type Client struct {
	UserAgent string
}

// setHeaders sets common headers for requests.
func (c Client) setHeaders(req *http.Request, isForm bool) {
	if isForm {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
}

// PostForm does a POST request with form data.
func (c Client) PostForm(url string, data io.Reader) (*http.Response, error) {
	client := http.Client{}

	req, err := http.NewRequest(http.MethodPost, url, data)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req, true)

	return client.Do(req)
}

// Post does a POST with the specified data.
func (c Client) Post(url string, data io.Reader) (*http.Response, error) {
	client := http.Client{}

	req, err := http.NewRequest(http.MethodPost, url, data)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req, false)

	return client.Do(req)
}
