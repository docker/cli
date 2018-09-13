package containerizedengine

import (
	"context"
	"fmt"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	clitypes "github.com/docker/cli/types"
	"github.com/docker/docker/api/types"
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
	return c.cclient.Install(ctx, image, containerd.WithInstallReplace, containerd.WithInstallPath("/usr"))
}
