package licenseutils

import (
	"context"

	"github.com/docker/licensing"
	"github.com/docker/licensing/model"
)

type (
	fakeLicensingClient struct {
		loginViaAuthFunc                 func(ctx context.Context, username, password string) (authToken string, err error)
		getHubUserOrgsFunc               func(ctx context.Context, authToken string) (orgs []model.Org, err error)
		getHubUserByNameFunc             func(ctx context.Context, username string) (user *model.User, err error)
		verifyLicenseFunc                func(ctx context.Context, license model.IssuedLicense) (res *model.CheckResponse, err error)
		generateNewTrialSubscriptionFunc func(ctx context.Context, authToken, dockerID, email string) (subscriptionID string, err error)
		listSubscriptionsFunc            func(ctx context.Context, authToken, dockerID string) (response []*model.Subscription, err error)
		listSubscriptionsDetailsFunc     func(ctx context.Context, authToken, dockerID string) (response []*model.SubscriptionDetail, err error)
		downloadLicenseFromHubFunc       func(ctx context.Context, authToken, subscriptionID string) (license *model.IssuedLicense, err error)
		parseLicenseFunc                 func(license []byte) (parsedLicense *model.IssuedLicense, err error)
		storeLicenseFunc                 func(ctx context.Context, dclnt licensing.WrappedDockerClient, licenses *model.IssuedLicense, localRootDir string) error
		loadLocalLicenseFunc             func(ctx context.Context, dclnt licensing.WrappedDockerClient) (*model.Subscription, error)
	}
)

func (c *fakeLicensingClient) LoginViaAuth(ctx context.Context, username, password string) (authToken string, err error) {
	if c.loginViaAuthFunc != nil {
		return c.loginViaAuthFunc(ctx, username, password)
	}
	return "", nil
}

func (c *fakeLicensingClient) GetHubUserOrgs(ctx context.Context, authToken string) (orgs []model.Org, err error) {
	if c.getHubUserOrgsFunc != nil {
		return c.getHubUserOrgsFunc(ctx, authToken)
	}
	return nil, nil
}

func (c *fakeLicensingClient) GetHubUserByName(ctx context.Context, username string) (user *model.User, err error) {
	if c.getHubUserByNameFunc != nil {
		return c.getHubUserByNameFunc(ctx, username)
	}
	return nil, nil
}

func (c *fakeLicensingClient) VerifyLicense(ctx context.Context, license model.IssuedLicense) (res *model.CheckResponse, err error) {
	if c.verifyLicenseFunc != nil {
		return c.verifyLicenseFunc(ctx, license)
	}
	return nil, nil
}

func (c *fakeLicensingClient) GenerateNewTrialSubscription(ctx context.Context, authToken, dockerID, email string) (subscriptionID string, err error) {
	if c.generateNewTrialSubscriptionFunc != nil {
		return c.generateNewTrialSubscriptionFunc(ctx, authToken, dockerID, email)
	}
	return "", nil
}

func (c *fakeLicensingClient) ListSubscriptions(ctx context.Context, authToken, dockerID string) (response []*model.Subscription, err error) {
	if c.listSubscriptionsFunc != nil {
		return c.listSubscriptionsFunc(ctx, authToken, dockerID)
	}
	return nil, nil
}

func (c *fakeLicensingClient) ListSubscriptionsDetails(ctx context.Context, authToken, dockerID string) (response []*model.SubscriptionDetail, err error) {
	if c.listSubscriptionsDetailsFunc != nil {
		return c.listSubscriptionsDetailsFunc(ctx, authToken, dockerID)
	}
	return nil, nil
}

func (c *fakeLicensingClient) DownloadLicenseFromHub(ctx context.Context, authToken, subscriptionID string) (license *model.IssuedLicense, err error) {
	if c.downloadLicenseFromHubFunc != nil {
		return c.downloadLicenseFromHubFunc(ctx, authToken, subscriptionID)
	}
	return nil, nil
}

func (c *fakeLicensingClient) ParseLicense(license []byte) (parsedLicense *model.IssuedLicense, err error) {
	if c.parseLicenseFunc != nil {
		return c.parseLicenseFunc(license)
	}
	return nil, nil
}

func (c *fakeLicensingClient) StoreLicense(ctx context.Context, dclnt licensing.WrappedDockerClient, licenses *model.IssuedLicense, localRootDir string) error {
	if c.storeLicenseFunc != nil {
		return c.storeLicenseFunc(ctx, dclnt, licenses, localRootDir)

	}
	return nil
}

func (c *fakeLicensingClient) LoadLocalLicense(ctx context.Context, dclnt licensing.WrappedDockerClient) (*model.Subscription, error) {

	if c.loadLocalLicenseFunc != nil {
		return c.loadLocalLicenseFunc(ctx, dclnt)

	}
	return nil, nil
}
