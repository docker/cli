package plugin

import (
	"context"
	"io"
	"net/http"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	pluginCreateFunc  func(createContext io.Reader, options client.PluginCreateOptions) (client.PluginCreateResult, error)
	pluginDisableFunc func(name string, options client.PluginDisableOptions) (client.PluginDisableResult, error)
	pluginEnableFunc  func(name string, options client.PluginEnableOptions) (client.PluginEnableResult, error)
	pluginRemoveFunc  func(name string, options client.PluginRemoveOptions) (client.PluginRemoveResult, error)
	pluginInstallFunc func(name string, options client.PluginInstallOptions) (client.PluginInstallResult, error)
	pluginListFunc    func(options client.PluginListOptions) (client.PluginListResult, error)
	pluginInspectFunc func(name string) (client.PluginInspectResult, error)
	pluginUpgradeFunc func(name string, options client.PluginUpgradeOptions) (client.PluginUpgradeResult, error)
}

func (c *fakeClient) PluginCreate(_ context.Context, createContext io.Reader, options client.PluginCreateOptions) (client.PluginCreateResult, error) {
	if c.pluginCreateFunc != nil {
		return c.pluginCreateFunc(createContext, options)
	}
	return client.PluginCreateResult{}, nil
}

func (c *fakeClient) PluginEnable(_ context.Context, name string, options client.PluginEnableOptions) (client.PluginEnableResult, error) {
	if c.pluginEnableFunc != nil {
		return c.pluginEnableFunc(name, options)
	}
	return client.PluginEnableResult{}, nil
}

func (c *fakeClient) PluginDisable(_ context.Context, name string, options client.PluginDisableOptions) (client.PluginDisableResult, error) {
	if c.pluginDisableFunc != nil {
		return c.pluginDisableFunc(name, options)
	}
	return client.PluginDisableResult{}, nil
}

func (c *fakeClient) PluginRemove(_ context.Context, name string, options client.PluginRemoveOptions) (client.PluginRemoveResult, error) {
	if c.pluginRemoveFunc != nil {
		return c.pluginRemoveFunc(name, options)
	}
	return client.PluginRemoveResult{}, nil
}

func (c *fakeClient) PluginInstall(_ context.Context, name string, options client.PluginInstallOptions) (client.PluginInstallResult, error) {
	if c.pluginInstallFunc != nil {
		return c.pluginInstallFunc(name, options)
	}
	return client.PluginInstallResult{}, nil
}

func (c *fakeClient) PluginList(_ context.Context, options client.PluginListOptions) (client.PluginListResult, error) {
	if c.pluginListFunc != nil {
		return c.pluginListFunc(options)
	}
	return client.PluginListResult{}, nil
}

func (c *fakeClient) PluginInspect(_ context.Context, name string, _ client.PluginInspectOptions) (client.PluginInspectResult, error) {
	if c.pluginInspectFunc != nil {
		return c.pluginInspectFunc(name)
	}
	return client.PluginInspectResult{}, nil
}

func (*fakeClient) Info(context.Context, client.InfoOptions) (client.SystemInfoResult, error) {
	return client.SystemInfoResult{}, nil
}

func (c *fakeClient) PluginUpgrade(_ context.Context, name string, options client.PluginUpgradeOptions) (client.PluginUpgradeResult, error) {
	if c.pluginUpgradeFunc != nil {
		return c.pluginUpgradeFunc(name, options)
	}
	// FIXME(thaJeztah): how to mock this?
	return http.NoBody, nil
}
