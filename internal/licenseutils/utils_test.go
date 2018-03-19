package licenseutils

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/licensing/model"
	"gotest.tools/assert"
)

func TestLoginNoAuth(t *testing.T) {
	ctx := context.Background()

	_, err := Login(ctx, &types.AuthConfig{})

	assert.ErrorContains(t, err, "must be logged in")
}

func TestGetOrgByID(t *testing.T) {
	orgs := []model.Org{
		{ID: "id1"},
		{ID: "id2"},
	}
	u := HubUser{
		Orgs: orgs,
	}
	o, err := u.GetOrgByID("id1")
	assert.NilError(t, err)
	assert.Assert(t, o.ID == "id1")
	o, err = u.GetOrgByID("id2")
	assert.NilError(t, err)
	assert.Assert(t, o.ID == "id2")
	o, err = u.GetOrgByID("id3")
	assert.ErrorContains(t, err, "not found")
}

func TestGetAvailableLicensesListFail(t *testing.T) {
	ctx := context.Background()
	user := HubUser{
		client: &fakeLicensingClient{
			listSubscriptionsFunc: func(ctx context.Context, authToken, dockerID string) (response []*model.Subscription, err error) {
				return nil, fmt.Errorf("list subscriptions error")
			},
		},
	}
	_, err := user.GetAvailableLicenses(ctx)
	assert.ErrorContains(t, err, "list subscriptions error")
}

func TestGetAvailableLicensesOrgFail(t *testing.T) {
	ctx := context.Background()
	user := HubUser{
		Orgs: []model.Org{
			{ID: "orgid"},
		},
		client: &fakeLicensingClient{
			listSubscriptionsFunc: func(ctx context.Context, authToken, dockerID string) (response []*model.Subscription, err error) {
				if dockerID == "orgid" {
					return nil, fmt.Errorf("list subscriptions org error")
				}
				return nil, nil
			},
		},
	}
	_, err := user.GetAvailableLicenses(ctx)
	assert.ErrorContains(t, err, "list subscriptions org error")
}

func TestGetAvailableLicensesHappy(t *testing.T) {
	ctx := context.Background()
	expiration := time.Now().Add(3600 * time.Second)
	user := HubUser{
		User: model.User{
			ID:       "userid",
			Username: "username",
		},
		Orgs: []model.Org{
			{
				ID:      "orgid",
				Orgname: "orgname",
			},
		},
		client: &fakeLicensingClient{
			listSubscriptionsFunc: func(ctx context.Context, authToken, dockerID string) (response []*model.Subscription, err error) {
				if dockerID == "orgid" {
					return []*model.Subscription{
						{
							State:   "expired",
							Expires: &expiration,
						},
						{
							State:    "active",
							DockerID: "orgid",
							Expires:  &expiration,
						},
						{
							State:    "active",
							DockerID: "invalidid",
							Expires:  &expiration,
						},
					}, nil
				} else if dockerID == "userid" {
					return []*model.Subscription{
						{
							State: "expired",
						},
						{
							State:    "active",
							DockerID: "userid",
							Expires:  &expiration,
							PricingComponents: model.PricingComponents{
								{
									Name:  "comp1",
									Value: 1,
								},
								{
									Name:  "comp2",
									Value: 2,
								},
							},
						},
					}, nil
				}
				return nil, nil
			},
		},
	}
	subs, err := user.GetAvailableLicenses(ctx)
	assert.NilError(t, err)
	assert.Assert(t, len(subs) == 3)
	assert.Assert(t, subs[0].Owner == "username")
	assert.Assert(t, subs[0].State == "active")
	assert.Assert(t, subs[0].ComponentsString == "comp1:1,comp2:2")
	assert.Assert(t, subs[1].Owner == "orgname")
	assert.Assert(t, subs[1].State == "active")
	assert.Assert(t, subs[2].Owner == "unknown")
	assert.Assert(t, subs[2].State == "active")
}

func TestGenerateTrialFail(t *testing.T) {
	ctx := context.Background()
	user := HubUser{
		client: &fakeLicensingClient{
			generateNewTrialSubscriptionFunc: func(ctx context.Context, authToken, dockerID, email string) (subscriptionID string, err error) {
				return "", fmt.Errorf("generate trial failure")
			},
		},
	}
	targetID := "targetidgoeshere"
	_, err := user.GenerateTrialLicense(ctx, targetID)
	assert.ErrorContains(t, err, "generate trial failure")
}

func TestGenerateTrialHappy(t *testing.T) {
	ctx := context.Background()
	user := HubUser{
		client: &fakeLicensingClient{
			generateNewTrialSubscriptionFunc: func(ctx context.Context, authToken, dockerID, email string) (subscriptionID string, err error) {
				return "subid", nil
			},
		},
	}
	targetID := "targetidgoeshere"
	_, err := user.GenerateTrialLicense(ctx, targetID)
	assert.NilError(t, err)
}

func TestGetIssuedLicense(t *testing.T) {
	ctx := context.Background()
	user := HubUser{
		client: &fakeLicensingClient{},
	}
	id := "idgoeshere"
	_, err := user.GetIssuedLicense(ctx, id)
	assert.NilError(t, err)
}

func TestLoadLocalIssuedLicenseNotExist(t *testing.T) {
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "licensing-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "subscription.lic")
	_, err = LoadLocalIssuedLicense(ctx, filename)
	assert.ErrorContains(t, err, "no such file")
}

func TestLoadLocalIssuedLicenseNotJson(t *testing.T) {
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "licensing-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "subscription.lic")
	err = ioutil.WriteFile(filename, []byte("not json"), 0644)
	assert.NilError(t, err)
	_, err = LoadLocalIssuedLicense(ctx, filename)
	assert.ErrorContains(t, err, "malformed license file")
}

func TestLoadLocalIssuedLicenseNoVerify(t *testing.T) {
	lclient := &fakeLicensingClient{
		verifyLicenseFunc: func(ctx context.Context, license model.IssuedLicense) (res *model.CheckResponse, err error) {
			return nil, fmt.Errorf("verification failed")
		},
	}
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "licensing-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "subscription.lic")
	err = ioutil.WriteFile(filename, []byte("{}"), 0644)
	assert.NilError(t, err)
	_, err = doLoadLocalIssuedLicense(ctx, filename, lclient)
	assert.ErrorContains(t, err, "verification failed")
}

func TestLoadLocalIssuedLicenseHappy(t *testing.T) {
	lclient := &fakeLicensingClient{}
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "licensing-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "subscription.lic")
	err = ioutil.WriteFile(filename, []byte("{}"), 0644)
	assert.NilError(t, err)
	_, err = doLoadLocalIssuedLicense(ctx, filename, lclient)
	assert.NilError(t, err)
}
