package licenseutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/licensing"
	"github.com/docker/licensing/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// HubUser wraps a licensing client and holds key information
// for a user to avoid multiple lookups
type HubUser struct {
	client licensing.Client
	token  string
	User   model.User
	Orgs   []model.Org
}

//GetOrgByID finds the org by the ID in the users list of orgs
func (u HubUser) GetOrgByID(orgID string) (model.Org, error) {
	for _, org := range u.Orgs {
		if org.ID == orgID {
			return org, nil
		}
	}
	return model.Org{}, fmt.Errorf("org %s not found", orgID)
}

func getClient() (licensing.Client, error) {
	baseURI, err := url.Parse(licensingDefaultBaseURI)
	if err != nil {
		return nil, err
	}

	return licensing.New(&licensing.Config{
		BaseURI:    *baseURI,
		HTTPClient: &http.Client{},
		PublicKeys: licensingPublicKeys,
	})
}

// Login to the license server and return a client that can be used to look up and download license files or generate new trial licenses
func Login(ctx context.Context, authConfig *types.AuthConfig) (HubUser, error) {
	lclient, err := getClient()
	if err != nil {
		return HubUser{}, err
	}

	// For licensing we know they must have a valid login session
	if authConfig.Username == "" {
		return HubUser{}, fmt.Errorf("you must be logged in to access licenses.  Please use 'docker login' then try again")
	}
	token, err := lclient.LoginViaAuth(ctx, authConfig.Username, authConfig.Password)
	if err != nil {
		return HubUser{}, err
	}
	user, err := lclient.GetHubUserByName(ctx, authConfig.Username)
	if err != nil {
		return HubUser{}, err
	}
	orgs, err := lclient.GetHubUserOrgs(ctx, token)
	if err != nil {
		return HubUser{}, err
	}
	return HubUser{
		client: lclient,
		token:  token,
		User:   *user,
		Orgs:   orgs,
	}, nil

}

// GetAvailableLicenses finds all available licenses for a given account and their orgs
func (u HubUser) GetAvailableLicenses(ctx context.Context) ([]LicenseDisplay, error) {
	subs, err := u.client.ListSubscriptions(ctx, u.token, u.User.ID)
	if err != nil {
		return nil, err
	}
	for _, org := range u.Orgs {
		orgSub, err := u.client.ListSubscriptions(ctx, u.token, org.ID)
		if err != nil {
			return nil, err
		}
		subs = append(subs, orgSub...)
	}

	// Convert the SubscriptionDetails to a more user-friendly type to render in the CLI

	res := []LicenseDisplay{}

	// Filter out expired licenses
	i := 0
	for _, s := range subs {
		if s.State == "active" && s.Expires != nil {
			owner := ""
			if s.DockerID == u.User.ID {
				owner = u.User.Username
			} else {
				ownerOrg, err := u.GetOrgByID(s.DockerID)
				if err == nil {
					owner = ownerOrg.Orgname
				} else {
					owner = "unknown"
					logrus.Debugf("Unable to lookup org ID %s: %s", s.DockerID, err)
				}
			}
			comps := []string{}
			for _, pc := range s.PricingComponents {
				comps = append(comps, fmt.Sprintf("%s:%d", pc.Name, pc.Value))
			}
			res = append(res, LicenseDisplay{
				Subscription:     *s,
				Num:              i,
				Owner:            owner,
				ComponentsString: strings.Join(comps, ","),
			})
			i++
		}
	}

	return res, nil
}

// GenerateTrialLicense will generate a new trial license for the specified user or org
func (u HubUser) GenerateTrialLicense(ctx context.Context, targetID string) (*model.IssuedLicense, error) {
	subID, err := u.client.GenerateNewTrialSubscription(ctx, u.token, targetID, u.User.Email)
	if err != nil {
		return nil, err
	}
	return u.client.DownloadLicenseFromHub(ctx, u.token, subID)
}

// GetIssuedLicense will download a license by ID
func (u HubUser) GetIssuedLicense(ctx context.Context, ID string) (*model.IssuedLicense, error) {
	return u.client.DownloadLicenseFromHub(ctx, u.token, ID)
}

// LoadLocalIssuedLicense will load a local license file
func LoadLocalIssuedLicense(ctx context.Context, filename string) (*model.IssuedLicense, error) {
	lclient, err := getClient()
	if err != nil {
		return nil, err
	}
	return doLoadLocalIssuedLicense(ctx, filename, lclient)
}

// GetLicenseSummary summarizes the license for the user
func GetLicenseSummary(ctx context.Context, license model.IssuedLicense) (string, error) {
	lclient, err := getClient()
	if err != nil {
		return "", err
	}

	cr, err := lclient.VerifyLicense(ctx, license)
	if err != nil {
		return "", err
	}
	return lclient.SummarizeLicense(cr, license.KeyID).String(), nil
}

func doLoadLocalIssuedLicense(ctx context.Context, filename string, lclient licensing.Client) (*model.IssuedLicense, error) {
	var license model.IssuedLicense
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// The file may contain a leading BOM, which will choke the
	// json deserializer.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	err = json.Unmarshal(data, &license)
	if err != nil {
		return nil, errors.Wrap(err, "malformed license file")
	}

	_, err = lclient.VerifyLicense(ctx, license)
	if err != nil {
		return nil, err
	}

	return &license, nil
}

// ApplyLicense will store a license on the local system
func ApplyLicense(ctx context.Context, dclient licensing.WrappedDockerClient, license *model.IssuedLicense) error {
	info, err := dclient.Info(ctx)
	if err != nil {
		return err
	}
	return licensing.StoreLicense(ctx, dclient, license, info.DockerRootDir)
}
