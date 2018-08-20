package engine

import (
	"context"

	"github.com/containerd/containerd"
	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/cli/internal/containerizedengine"
	"github.com/docker/docker/api/types"
)

type (
	fakeContainerizedEngineClient struct {
		closeFunc          func() error
		activateEngineFunc func(ctx context.Context,
			opts containerizedengine.EngineInitOptions,
			out containerizedengine.OutStream,
			authConfig *types.AuthConfig,
			healthfn func(context.Context) error) error
		initEngineFunc func(ctx context.Context,
			opts containerizedengine.EngineInitOptions,
			out containerizedengine.OutStream,
			authConfig *types.AuthConfig,
			healthfn func(context.Context) error) error
		doUpdateFunc func(ctx context.Context,
			opts containerizedengine.EngineInitOptions,
			out containerizedengine.OutStream,
			authConfig *types.AuthConfig,
			healthfn func(context.Context) error) error
		getEngineVersionsFunc func(ctx context.Context,
			registryClient registryclient.RegistryClient,
			currentVersion,
			imageName string) (containerizedengine.AvailableVersions, error)

		getEngineFunc               func(ctx context.Context) (containerd.Container, error)
		removeEngineFunc            func(ctx context.Context, engine containerd.Container) error
		getCurrentEngineVersionFunc func(ctx context.Context) (containerizedengine.EngineInitOptions, error)
	}
)

func (w *fakeContainerizedEngineClient) Close() error {
	if w.closeFunc != nil {
		return w.closeFunc()
	}
	return nil
}

func (w *fakeContainerizedEngineClient) ActivateEngine(ctx context.Context,
	opts containerizedengine.EngineInitOptions,
	out containerizedengine.OutStream,
	authConfig *types.AuthConfig,
	healthfn func(context.Context) error) error {
	if w.activateEngineFunc != nil {
		return w.activateEngineFunc(ctx, opts, out, authConfig, healthfn)
	}
	return nil
}
func (w *fakeContainerizedEngineClient) InitEngine(ctx context.Context,
	opts containerizedengine.EngineInitOptions,
	out containerizedengine.OutStream,
	authConfig *types.AuthConfig,
	healthfn func(context.Context) error) error {
	if w.initEngineFunc != nil {
		return w.initEngineFunc(ctx, opts, out, authConfig, healthfn)
	}
	return nil
}
func (w *fakeContainerizedEngineClient) DoUpdate(ctx context.Context,
	opts containerizedengine.EngineInitOptions,
	out containerizedengine.OutStream,
	authConfig *types.AuthConfig,
	healthfn func(context.Context) error) error {
	if w.doUpdateFunc != nil {
		return w.doUpdateFunc(ctx, opts, out, authConfig, healthfn)
	}
	return nil
}
func (w *fakeContainerizedEngineClient) GetEngineVersions(ctx context.Context,
	registryClient registryclient.RegistryClient,
	currentVersion, imageName string) (containerizedengine.AvailableVersions, error) {

	if w.getEngineVersionsFunc != nil {
		return w.getEngineVersionsFunc(ctx, registryClient, currentVersion, imageName)
	}
	return containerizedengine.AvailableVersions{}, nil
}

func (w *fakeContainerizedEngineClient) GetEngine(ctx context.Context) (containerd.Container, error) {
	if w.getEngineFunc != nil {
		return w.getEngineFunc(ctx)
	}
	return nil, nil
}
func (w *fakeContainerizedEngineClient) RemoveEngine(ctx context.Context, engine containerd.Container) error {
	if w.removeEngineFunc != nil {
		return w.removeEngineFunc(ctx, engine)
	}
	return nil
}
func (w *fakeContainerizedEngineClient) GetCurrentEngineVersion(ctx context.Context) (containerizedengine.EngineInitOptions, error) {
	if w.getCurrentEngineVersionFunc != nil {
		return w.getCurrentEngineVersionFunc(ctx)
	}
	return containerizedengine.EngineInitOptions{}, nil
}
