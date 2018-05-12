package plugin

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	pluginCreateFunc  func(createContext io.Reader, createOptions types.PluginCreateOptions) error
	pluginDisableFunc func(name string, disableOptions types.PluginDisableOptions) error
	pluginEnableFunc  func(name string, options types.PluginEnableOptions) error
	pluginRemoveFunc  func(name string, options types.PluginRemoveOptions) error
}

func (c *fakeClient) PluginCreate(ctx context.Context, createContext io.Reader, createOptions types.PluginCreateOptions) error {
	if c.pluginCreateFunc != nil {
		return c.pluginCreateFunc(createContext, createOptions)
	}
	return nil
}

func (c *fakeClient) PluginEnable(ctx context.Context, name string, enableOptions types.PluginEnableOptions) error {
	if c.pluginEnableFunc != nil {
		return c.pluginEnableFunc(name, enableOptions)
	}
	return nil
}

func (c *fakeClient) PluginDisable(context context.Context, name string, disableOptions types.PluginDisableOptions) error {
	if c.pluginDisableFunc != nil {
		return c.pluginDisableFunc(name, disableOptions)
	}
	return nil
}

func (c *fakeClient) PluginRemove(context context.Context, name string, removeOptions types.PluginRemoveOptions) error {
	if c.pluginRemoveFunc != nil {
		return c.pluginRemoveFunc(name, removeOptions)
	}
	return nil
}
