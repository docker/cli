docker/licensing
=========

## Overview

*licensing* is a library for interacting with Docker issued product licenses. It facilitates user's authentication to the [Docker Hub](https://hub.docker.com), provides a mechanism for retrieving a user's existing docker-issued subscriptions/licenses, detects and verifies locally stored licenses, and can be used to provision trial licenses for [Docker Enterprise Edition](https://www.docker.com/enterprise-edition).

License
=========
docker/licensing is licensed under the Apache License, Version 2.0. See
[LICENSE](https://github.com/docker/licensing/blob/master/LICENSE) for the full
license text.

Usage
========
```go
package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/docker/licensing"
	"github.com/docker/licensing/model"
)

const (
	hubURL      = "https://hub.docker.com"
	pubKey      = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0Ka2lkOiBKN0xEOjY3VlI6TDVIWjpVN0JBOjJPNEc6NEFMMzpPRjJOOkpIR0I6RUZUSDo1Q1ZROk1GRU86QUVJVAoKTUlJQ0lqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FnOEFNSUlDQ2dLQ0FnRUF5ZEl5K2xVN283UGNlWSs0K3MrQwpRNU9FZ0N5RjhDeEljUUlXdUs4NHBJaVpjaVk2NzMweUNZbndMU0tUbHcrVTZVQy9RUmVXUmlvTU5ORTVEczVUCllFWGJHRzZvbG0ycWRXYkJ3Y0NnKzJVVUgvT2NCOVd1UDZnUlBIcE1GTXN4RHpXd3ZheThKVXVIZ1lVTFVwbTEKSXYrbXE3bHA1blEvUnhyVDBLWlJBUVRZTEVNRWZHd20zaE1PL2dlTFBTK2hnS1B0SUhsa2c2L1djb3hUR29LUAo3OWQvd2FIWXhHTmw3V2hTbmVpQlN4YnBiUUFLazIxbGc3OThYYjd2WnlFQVRETXJSUjlNZUU2QWRqNUhKcFkzCkNveVJBUENtYUtHUkNLNHVvWlNvSXUwaEZWbEtVUHliYncwMDBHTyt3YTJLTjhVd2dJSW0waTVJMXVXOUdrcTQKempCeTV6aGdxdVVYYkc5YldQQU9ZcnE1UWE4MUR4R2NCbEp5SFlBcCtERFBFOVRHZzR6WW1YakpueFpxSEVkdQpHcWRldlo4WE1JMHVrZmtHSUkxNHdVT2lNSUlJclhsRWNCZi80Nkk4Z1FXRHp4eWNaZS9KR1grTEF1YXlYcnlyClVGZWhWTlVkWlVsOXdYTmFKQitrYUNxejVRd2FSOTNzR3crUVNmdEQwTnZMZTdDeU9IK0U2dmc2U3QvTmVUdmcKdjhZbmhDaVhJbFo4SE9mSXdOZTd0RUYvVWN6NU9iUHlrbTN0eWxyTlVqdDBWeUFtdHRhY1ZJMmlHaWhjVVBybQprNGxWSVo3VkQvTFNXK2k3eW9TdXJ0cHNQWGNlMnBLRElvMzBsSkdoTy8zS1VtbDJTVVpDcXpKMXlFbUtweXNICjVIRFc5Y3NJRkNBM2RlQWpmWlV2TjdVQ0F3RUFBUT09Ci0tLS0tRU5EIFBVQkxJQyBLRVktLS0tLQo="
	username    = "docker username"
	password    = "your password"
	appFeature  = "jump"
)

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	hubURI, err := url.Parse(hubURL)
	panicOnErr(err)

	// setup client
	c, err := licensing.New(&licensing.Config{
		BaseURI:    *hubURI,
		HTTPClient: nil,
		PublicKeys: []string{pubKey},
	})
	panicOnErr(err)

	// grab token
	ctx := context.Background()
	token, err := c.LoginViaAuth(ctx, username, password)
	panicOnErr(err)

	// fetch dockerID, if not already known
	id, err := c.GetHubUserByName(ctx, username)
	panicOnErr(err)

	subs, err := c.ListSubscriptions(ctx, token, id.ID)
	panicOnErr(err)

	// find first available subscription with given feature
	var featuredSub *model.Subscription
	for _, sub := range subs {
		_, ok := sub.GetFeatureValue(appFeature)
		if ok {
			featuredSub = sub
			break
		}
	}
	if featuredSub == nil {
		fmt.Println("account has no subscriptions with the desired feature entitlements")
		return
	}

	// download license file for this subscription
	subLic, err := c.DownloadLicenseFromHub(ctx, token, featuredSub.ID)
	panicOnErr(err)

	// verify license is issued by corresponding keypair and is not expired
	licFile, err := c.VerifyLicense(ctx, *subLic)
	panicOnErr(err)

	fmt.Println("license summary: ", c.SummarizeLicense(licFile))
}
```
