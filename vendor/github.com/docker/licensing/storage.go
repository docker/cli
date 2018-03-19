package licensing

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/licensing/model"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
)

var (
	licenseNamePrefix = "com.docker.license"
	licenseFilename   = "docker.lic"

	// ErrWorkerNode returned on a swarm worker node - lookup licenses on swarm managers
	ErrWorkerNode = fmt.Errorf("this node is not a swarm manager - check license status on a manager node")
	// ErrUnlicensed returned when no license found
	ErrUnlicensed = fmt.Errorf("no license found")
)

// WrappedDockerClient provides methods useful for installing licenses to the wrapped docker engine or cluster
type WrappedDockerClient interface {
	Info(ctx context.Context) (types.Info, error)
	NodeList(ctx context.Context, options types.NodeListOptions) ([]swarm.Node, error)
	ConfigCreate(ctx context.Context, config swarm.ConfigSpec) (types.ConfigCreateResponse, error)
	ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error)
	ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error)
}

// StoreLicense will store the license on the host filesystem and swarm (if swarm is active)
func StoreLicense(ctx context.Context, clnt WrappedDockerClient, license *model.IssuedLicense, rootDir string) error {

	licenseData, err := json.Marshal(*license)
	if err != nil {
		return err
	}

	// First determine if we're in swarm-mode or a stand-alone engine
	_, err = clnt.NodeList(ctx, types.NodeListOptions{})
	if err != nil { // TODO - check for the specific error message
		return writeLicenseToHost(ctx, clnt, licenseData, rootDir)
	}
	// Load this in the latest license index
	latestVersion, err := getLatestNamedConfig(clnt, licenseNamePrefix)
	if err != nil {
		return fmt.Errorf("unable to get latest license version: %s", err)
	}
	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name: fmt.Sprintf("%s-%d", licenseNamePrefix, latestVersion+1),
			Labels: map[string]string{
				"com.docker.ucp.access.label":     "/",
				"com.docker.ucp.collection":       "swarm",
				"com.docker.ucp.collection.root":  "true",
				"com.docker.ucp.collection.swarm": "true",
			},
		},
		Data: licenseData,
	}
	_, err = clnt.ConfigCreate(context.Background(), spec)
	if err != nil {

		return fmt.Errorf("Failed to create license: %s", err)
	}

	return nil
}

func (c *client) LoadLocalLicense(ctx context.Context, clnt WrappedDockerClient) (*model.Subscription, error) {
	info, err := clnt.Info(ctx)
	if err != nil {
		return nil, err
	}

	var licenseData []byte
	if info.Swarm.LocalNodeState != "active" {
		licenseData, err = readLicenseFromHost(ctx, info.DockerRootDir)
	} else {
		// Load the latest license index
		latestVersion, err := getLatestNamedConfig(clnt, licenseNamePrefix)
		if err != nil {
			if strings.Contains(err.Error(), "not a swarm manager.") {
				return nil, ErrWorkerNode
			}
			return nil, fmt.Errorf("unable to get latest license version: %s", err)
		}
		cfg, _, err := clnt.ConfigInspectWithRaw(ctx, fmt.Sprintf("%s-%d", licenseNamePrefix, latestVersion))
		if err != nil {
			return nil, fmt.Errorf("unable to load license from swarm config: %s", err)
		}
		licenseData = cfg.Spec.Data
	}
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrUnlicensed
		}
		return nil, fmt.Errorf("Failed to create license: %s", err)
	}

	parsedLicense, err := c.ParseLicense(licenseData)
	if err != nil {
		return nil, err
	}
	checkResponse, err := c.VerifyLicense(ctx, *parsedLicense)
	if err != nil {
		return nil, err
	}

	// TODO - this translation still needs some work
	// Primary missing piece is how to distinguish from basic, vs std/advanced
	var productID string
	var ratePlan string
	var state string
	switch strings.ToLower(checkResponse.Tier) {
	case "internal":
		productID = "docker-ee-trial"
		ratePlan = "free-trial"
	case "production":
		productID = "docker-ee"
		if checkResponse.ScanningEnabled {
			ratePlan = "nfr-advanced"
		} else {
			ratePlan = "nfr-standard"
		}
	}

	// Determine if the license has already expired
	if checkResponse.Expiration.Before(time.Now()) {
		state = "expired"
	} else {
		state = "active"
	}

	// Translate the legacy structure into the new Subscription fields
	return &model.Subscription{
		// Name
		ID: parsedLicense.KeyID, // This is not actually the same, but is unique
		// DockerID
		ProductID:       productID,
		ProductRatePlan: ratePlan,
		// ProductRatePlanID
		// Start
		Expires: &checkResponse.Expiration,
		State:   state,
		// Eusa
		PricingComponents: model.PricingComponents{
			{
				Name:  "Nodes",
				Value: checkResponse.MaxEngines,
			},
		},
	}, nil
}

// getLatestNamedConfig looks for versioned instances of configs with the
// given name prefix which have a `-NUM` integer version suffix. Returns the
// config with the higest version number found or nil if no such configs exist
// along with its version number.
func getLatestNamedConfig(dclient WrappedDockerClient, namePrefix string) (int, error) {
	latestVersion := -1
	// List any/all existing configs so that we create a newer version than
	// any that already exist.
	filter := filters.NewArgs()
	filter.Add("name", namePrefix)
	existingConfigs, err := dclient.ConfigList(context.Background(), types.ConfigListOptions{Filters: filter})
	if err != nil {
		return latestVersion, fmt.Errorf("unable to list existing configs: %s", err)
	}

	for _, existingConfig := range existingConfigs {
		existingConfigName := existingConfig.Spec.Name
		nameSuffix := strings.TrimPrefix(existingConfigName, namePrefix)
		if nameSuffix == "" || nameSuffix[0] != '-' {
			continue // No version specifier?
		}

		versionSuffix := nameSuffix[1:] // Trim the version separator.
		existingVersion, err := strconv.Atoi(versionSuffix)
		if err != nil {
			continue // Unable to parse version as integer.
		}
		if existingVersion > latestVersion {
			latestVersion = existingVersion
		}
	}

	return latestVersion, nil
}

func writeLicenseToHost(ctx context.Context, dclient WrappedDockerClient, license []byte, rootDir string) error {
	// TODO we should write the file out over the clnt instead of to the local filesystem
	return ioutil.WriteFile(filepath.Join(rootDir, licenseFilename), license, 0644)
}

func readLicenseFromHost(ctx context.Context, rootDir string) ([]byte, error) {
	// TODO we should read the file in over the clnt instead of to the local filesystem
	return ioutil.ReadFile(filepath.Join(rootDir, licenseFilename))
}
