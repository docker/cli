package containerizedengine

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	clitypes "github.com/docker/cli/types"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	ver "github.com/hashicorp/go-version"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// ActivateEngine will switch the image from the CE to EE image
func (c *baseClient) ActivateEngine(ctx context.Context, opts clitypes.EngineInitOptions, out clitypes.OutStream,
	authConfig *types.AuthConfig, healthfn func(context.Context) error) error {

	ctx = namespaces.WithNamespace(ctx, engineNamespace)
	return c.DoUpdate(ctx, opts, out, authConfig, healthfn)
}

// DoUpdate performs the underlying engine update
func (c *baseClient) DoUpdate(ctx context.Context, opts clitypes.EngineInitOptions, out clitypes.OutStream,
	authConfig *types.AuthConfig, healthfn func(context.Context) error) error {

	ctx = namespaces.WithNamespace(ctx, engineNamespace)
	if opts.EngineVersion == "" {
		// TODO - Future enhancement: This could be improved to be
		// smart about figuring out the latest patch rev for the
		// current engine version and automatically apply it so users
		// could stay in sync by simply having a scheduled
		// `docker engine update`
		return fmt.Errorf("please pick the version you want to update to")
	}

	localMetadata, err := c.GetCurrentRuntimeMetadata(ctx, "")
	if err == nil {
		if opts.EngineImage == "" {
			if strings.Contains(strings.ToLower(localMetadata.Platform), "enterprise") {
				opts.EngineImage = "engine-enterprise"
			} else {
				opts.EngineImage = "engine-community"
			}
		}
	}
	if opts.EngineImage == "" {
		return fmt.Errorf("please pick the engine image to update with (engine-community or engine-enterprise)")
	}

	imageName := fmt.Sprintf("%s/%s:%s", opts.RegistryPrefix, opts.EngineImage, opts.EngineVersion)

	// Look for desired image
	image, err := c.cclient.GetImage(ctx, imageName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			image, err = c.pullWithAuth(ctx, imageName, out, authConfig)
			if err != nil {
				return errors.Wrapf(err, "unable to pull image %s", imageName)
			}
		} else {
			return errors.Wrapf(err, "unable to check for image %s", imageName)
		}
	}

	// Make sure we're safe to proceed
	newMetadata, err := c.PreflightCheck(ctx, image)
	if err != nil {
		return err
	}
	// Grab current metadata for comparison purposes
	if localMetadata != nil {
		if localMetadata.Platform != newMetadata.Platform {
			fmt.Fprintf(out, "\nNotice: you have switched to \"%s\".  Please refer to %s for update instructions.\n\n", newMetadata.Platform, c.GetReleaseNotesURL(imageName))
		}
	}

	err = c.cclient.Install(ctx, image, containerd.WithInstallReplace, containerd.WithInstallPath("/usr"))
	if err != nil {
		return err
	}

	return c.WriteRuntimeMetadata(ctx, "", newMetadata)
}

var defaultDockerRoot = "/var/lib/docker"

// GetCurrentRuntimeMetadata loads the current daemon runtime metadata information from the local host
func (c *baseClient) GetCurrentRuntimeMetadata(_ context.Context, dockerRoot string) (*RuntimeMetadata, error) {
	if dockerRoot == "" {
		dockerRoot = defaultDockerRoot
	}
	filename := filepath.Join(dockerRoot, RuntimeMetadataName+".json")

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var res RuntimeMetadata
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, errors.Wrapf(err, "malformed runtime metadata file %s", filename)
	}
	return &res, nil
}

func (c *baseClient) WriteRuntimeMetadata(_ context.Context, dockerRoot string, metadata *RuntimeMetadata) error {
	if dockerRoot == "" {
		dockerRoot = defaultDockerRoot
	}
	filename := filepath.Join(dockerRoot, RuntimeMetadataName+".json")

	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// PreflightCheck verifies the specified image is compatible with the local system before proceeding to update/activate
// If things look good, the RuntimeMetadata for the new image is returned and can be written out to the host
func (c *baseClient) PreflightCheck(ctx context.Context, image containerd.Image) (*RuntimeMetadata, error) {
	var metadata RuntimeMetadata
	ic, err := image.Config(ctx)
	if err != nil {
		return nil, err
	}
	var (
		ociimage v1.Image
		config   v1.ImageConfig
	)
	switch ic.MediaType {
	case v1.MediaTypeImageConfig, images.MediaTypeDockerSchema2Config:
		p, err := content.ReadBlob(ctx, image.ContentStore(), ic)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(p, &ociimage); err != nil {
			return nil, err
		}
		config = ociimage.Config
	default:
		return nil, fmt.Errorf("unknown image %s config media type %s", image.Name(), ic.MediaType)
	}

	metadataString, ok := config.Labels[RuntimeMetadataName]
	if !ok {
		return nil, fmt.Errorf("image %s does not contain runtime metadata label %s", image.Name(), RuntimeMetadataName)
	}
	err = json.Unmarshal([]byte(metadataString), &metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "malformed runtime metadata file in %s", image.Name())
	}

	// Current CLI only supports host install runtime
	if metadata.Runtime != "host_install" {
		return nil, fmt.Errorf("unsupported runtime: %s\nPlease consult the release notes at %s for upgrade instructions", metadata.Runtime, c.GetReleaseNotesURL(image.Name()))
	}

	// Verify local containerd is new enough
	localVersion, err := c.cclient.Version(ctx)
	if err != nil {
		return nil, err
	}
	if metadata.ContainerdMinVersion != "" {
		lv, err := ver.NewVersion(localVersion.Version)
		if err != nil {
			return nil, err
		}
		mv, err := ver.NewVersion(metadata.ContainerdMinVersion)
		if err != nil {
			return nil, err
		}
		if lv.LessThan(mv) {
			return nil, fmt.Errorf("local containerd is too old: %s - this engine version requires %s or newer.\nPlease consult the release notes at %s for upgrade instructions",
				localVersion.Version, metadata.ContainerdMinVersion, c.GetReleaseNotesURL(image.Name()))
		}
	} // If omitted on metadata, no hard dependency on containerd version beyond 18.09 baseline

	// All checks look OK, proceed with update
	return &metadata, nil
}

// GetReleaseNotesURL returns a release notes url
// If the image name does not contain a version tag, the base release notes URL is returned
func (c *baseClient) GetReleaseNotesURL(imageName string) string {
	versionTag := ""
	distributionRef, err := reference.ParseNormalizedNamed(imageName)
	if err == nil {
		taggedRef, ok := distributionRef.(reference.NamedTagged)
		if ok {
			versionTag = taggedRef.Tag()
		}
	}
	return fmt.Sprintf("%s/%s", ReleaseNotePrefix, versionTag)
}
