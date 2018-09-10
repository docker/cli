package licensing

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/docker/libtrust"
	"github.com/docker/licensing/lib/errors"
	"github.com/docker/licensing/lib/go-auth/jwt"
	"github.com/docker/licensing/lib/go-clientlib"
	"github.com/docker/licensing/model"
)

const (
	trialProductID  = "docker-ee-trial"
	trialRatePlanID = "free-trial"
)

// Client represents the licensing package interface, including methods for authentication and interaction with Docker
// licensing, accounts, and billing services
type Client interface {
	LoginViaAuth(ctx context.Context, username, password string) (authToken string, err error)
	GetHubUserOrgs(ctx context.Context, authToken string) (orgs []model.Org, err error)
	GetHubUserByName(ctx context.Context, username string) (user *model.User, err error)
	VerifyLicense(ctx context.Context, license model.IssuedLicense) (res *model.CheckResponse, err error)
	GenerateNewTrialSubscription(ctx context.Context, authToken, dockerID, email string) (subscriptionID string, err error)
	ListSubscriptions(ctx context.Context, authToken, dockerID string) (response []*model.Subscription, err error)
	ListSubscriptionsDetails(ctx context.Context, authToken, dockerID string) (response []*model.SubscriptionDetail, err error)
	DownloadLicenseFromHub(ctx context.Context, authToken, subscriptionID string) (license *model.IssuedLicense, err error)
	ParseLicense(license []byte) (parsedLicense *model.IssuedLicense, err error)
	StoreLicense(ctx context.Context, dclnt WrappedDockerClient, licenses *model.IssuedLicense, localRootDir string) error
	LoadLocalLicense(ctx context.Context, dclnt WrappedDockerClient) (*model.Subscription, error)
	SummarizeLicense(res *model.CheckResponse, keyID string) *model.Subscription
}

func (c *client) LoginViaAuth(ctx context.Context, username, password string) (string, error) {
	creds, err := c.login(ctx, username, password)
	if err != nil {
		return "", errors.Wrap(err, errors.Fields{
			"username": username,
		})
	}

	return creds.Token, nil
}

func (c *client) GetHubUserOrgs(ctx context.Context, authToken string) ([]model.Org, error) {
	ctx = jwt.NewContext(ctx, authToken)

	orgs, err := c.getUserOrgs(ctx, model.PaginationParams{})
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to get orgs for user")
	}

	return orgs, nil
}

func (c *client) GetHubUserByName(ctx context.Context, username string) (*model.User, error) {
	user, err := c.getUserByName(ctx, username)
	if err != nil {
		return nil, errors.Wrap(err, errors.Fields{
			"username": username,
		})
	}

	return user, nil
}

func (c *client) VerifyLicense(ctx context.Context, license model.IssuedLicense) (*model.CheckResponse, error) {
	res, err := c.check(ctx, license)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to verify license")
	}

	return res, nil
}

func (c *client) GenerateNewTrialSubscription(ctx context.Context, authToken, dockerID, email string) (string, error) {
	ctx = jwt.NewContext(ctx, authToken)

	if _, err := c.getAccount(ctx, dockerID); err != nil {
		code, ok := errors.HTTPStatus(err)
		// create billing account if one is not found
		if ok && code == http.StatusNotFound {
			_, err = c.createAccount(ctx, dockerID, &model.AccountCreationRequest{
				Profile: model.Profile{
					Email: email,
				},
			})
			if err != nil {
				return "", errors.Wrap(err, errors.Fields{
					"dockerID": dockerID,
					"email":    email,
				})
			}
		} else {
			return "", errors.Wrap(err, errors.Fields{
				"dockerID": dockerID,
			})
		}
	}

	sub, err := c.createSubscription(ctx, &model.SubscriptionCreationRequest{
		Name:            "Docker Enterprise Free Trial",
		DockerID:        dockerID,
		ProductID:       trialProductID,
		ProductRatePlan: trialRatePlanID,
		Eusa: &model.EusaState{
			Accepted: true,
		},
	})
	if err != nil {
		return "", errors.Wrap(err, errors.Fields{
			"dockerID": dockerID,
			"email":    email,
		})
	}

	return sub.ID, nil
}

// ListSubscriptions returns basic descriptions of all subscriptions to docker enterprise products for the given dockerID
func (c *client) ListSubscriptions(ctx context.Context, authToken, dockerID string) ([]*model.Subscription, error) {
	ctx = jwt.NewContext(ctx, authToken)

	subs, err := c.listSubscriptions(ctx, map[string]string{"docker_id": dockerID})
	if err != nil {
		return nil, errors.Wrap(err, errors.Fields{
			"dockerID": dockerID,
		})
	}

	// filter out non docker licenses
	dockerSubs := []*model.Subscription{}
	for _, sub := range subs {
		if !strings.HasPrefix(sub.ProductID, "docker-ee") {
			continue
		}

		dockerSubs = append(dockerSubs, sub)
	}

	return dockerSubs, nil
}

// ListDetailedSubscriptions returns detailed subscriptions to docker enterprise products for the given dockerID
func (c *client) ListSubscriptionsDetails(ctx context.Context, authToken, dockerID string) ([]*model.SubscriptionDetail, error) {
	ctx = jwt.NewContext(ctx, authToken)

	subs, err := c.listSubscriptionsDetails(ctx, map[string]string{"docker_id": dockerID})
	if err != nil {
		return nil, errors.Wrap(err, errors.Fields{
			"dockerID": dockerID,
		})
	}

	// filter out non docker licenses
	dockerSubs := []*model.SubscriptionDetail{}
	for _, sub := range subs {
		if !strings.HasPrefix(sub.ProductID, "docker-ee") {
			continue
		}

		dockerSubs = append(dockerSubs, sub)
	}

	return dockerSubs, nil
}

func (c *client) DownloadLicenseFromHub(ctx context.Context, authToken, subscriptionID string) (*model.IssuedLicense, error) {
	ctx = jwt.NewContext(ctx, authToken)

	license, err := c.getLicenseFile(ctx, subscriptionID)
	if err != nil {
		return nil, errors.Wrap(err, errors.Fields{
			"subscriptionID": subscriptionID,
		})
	}

	return license, nil
}

func (c *client) ParseLicense(license []byte) (*model.IssuedLicense, error) {
	parsedLicense := &model.IssuedLicense{}
	// The file may contain a leading BOM, which will choke the
	// json deserializer.
	license = bytes.Trim(license, "\xef\xbb\xbf")

	if err := json.Unmarshal(license, &parsedLicense); err != nil {
		return nil, errors.WithMessage(err, "failed to parse license")
	}

	return parsedLicense, nil
}

type client struct {
	publicKeys []libtrust.PublicKey
	hclient    *http.Client
	baseURI    url.URL
}

// Config holds licensing client configuration
type Config struct {
	BaseURI    url.URL
	HTTPClient *http.Client
	// used by licensing client to validate an issued license
	PublicKeys []string
}

func errorSummary(body []byte) string {
	var be struct {
		Message string `json:"message"`
	}

	jsonErr := json.Unmarshal(body, &be)
	if jsonErr != nil {
		return clientlib.DefaultErrorSummary(body)
	}

	return be.Message
}

// New creates a new licensing Client
func New(config *Config) (Client, error) {
	publicKeys, err := unmarshalPublicKeys(config.PublicKeys)
	if err != nil {
		return nil, err
	}

	hclient := config.HTTPClient
	if hclient == nil {
		hclient = &http.Client{}
	}

	return &client{
		baseURI:    config.BaseURI,
		hclient:    hclient,
		publicKeys: publicKeys,
	}, nil
}

func unmarshalPublicKeys(publicKeys []string) ([]libtrust.PublicKey, error) {
	trustKeys := make([]libtrust.PublicKey, len(publicKeys))

	for i, publicKey := range publicKeys {
		trustKey, err := unmarshalPublicKey(publicKey)
		if err != nil {
			return nil, err
		}

		trustKeys[i] = trustKey
	}
	return trustKeys, nil
}

func unmarshalPublicKey(publicKey string) (libtrust.PublicKey, error) {
	pemBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return nil, errors.Wrapf(err, errors.Fields{
			"public_key": publicKey,
		}, "decode public key failed")
	}

	key, err := libtrust.UnmarshalPublicKeyPEM(pemBytes)
	if err != nil {
		return nil, errors.Wrapf(err, errors.Fields{
			"public_key": publicKey,
		}, "unmarshal public key failed")
	}
	return key, nil
}

func (c *client) doReq(ctx context.Context, method string, url *url.URL, opts ...clientlib.RequestOption) (*http.Request, *http.Response, error) {
	return clientlib.Do(ctx, method, url.String(), append(c.requestDefaults(), opts...)...)
}

func (c *client) doRequestNoAuth(ctx context.Context, method string, url *url.URL, opts ...clientlib.RequestOption) (*http.Request, *http.Response, error) {
	return clientlib.Do(ctx, method, url.String(), append(c.requestDefaults(), opts...)...)
}

func (c *client) requestDefaults() []clientlib.RequestOption {
	return []clientlib.RequestOption{
		func(req *clientlib.Request) {
			tok, _ := jwt.FromContext(req.Context())
			req.Header.Add("Authorization", "Bearer "+tok)
			req.ErrorSummary = errorSummary
			req.Client = c.hclient
		},
	}
}

func (c *client) StoreLicense(ctx context.Context, dclnt WrappedDockerClient, licenses *model.IssuedLicense, localRootDir string) error {
	return StoreLicense(ctx, dclnt, licenses, localRootDir)
}
