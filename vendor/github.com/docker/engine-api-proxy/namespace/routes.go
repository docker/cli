package pipeline

import (
	"github.com/docker/docker/client"
	"github.com/docker/engine-api-proxy/proxy"
	"github.com/docker/engine-api-proxy/routes"
)

// MiddlewareRoutes returns the pipeline middleware routes.
func MiddlewareRoutes(client client.APIClient, lookup LookupScope) []proxy.MiddlewareRoute {
	scopedContainerRoute := func(route *routes.Route) proxy.MiddlewareRoute {
		return newScopeContainerPath(route, lookup, client)
	}

	return []proxy.MiddlewareRoute{
		// Container routes
		newContainerCreateRoute(lookup),
		newContainerListRoute(lookup),
		newContainerInspectRoute(lookup, client),
		newContainerCommitRoute(lookup),
		scopedContainerRoute(routes.ContainerArchiveGet),
		scopedContainerRoute(routes.ContainerArchiveHead),
		scopedContainerRoute(routes.ContainerArchivePut),
		scopedContainerRoute(routes.ContainerAttach),
		scopedContainerRoute(routes.ContainerAttachWS),
		scopedContainerRoute(routes.ContainerRemove),
		scopedContainerRoute(routes.ContainerKill),
		scopedContainerRoute(routes.ContainerPause),
		scopedContainerRoute(routes.ContainerRename),
		scopedContainerRoute(routes.ContainerResize),
		scopedContainerRoute(routes.ContainerRestart),
		scopedContainerRoute(routes.ContainerStart),
		scopedContainerRoute(routes.ContainerStats),
		scopedContainerRoute(routes.ContainerStop),
		scopedContainerRoute(routes.ContainerTop),
		scopedContainerRoute(routes.ContainerUnpause),
		scopedContainerRoute(routes.ContainerUpdate),
		scopedContainerRoute(routes.ContainerWait),
		scopedContainerRoute(routes.ContainerLogs),
		scopedContainerRoute(routes.ContainerChanges),
		scopedContainerRoute(routes.ContainerExport),
		scopedContainerRoute(routes.ContainerExecCreate),

		// Volume routes
		newVolumeListRoute(lookup),
		newObjectInspectRoute(routes.VolumeInspect, newVolumeScopePath, lookup, client),
		newVolumeCreateRoute(lookup),
		newScopeObjectPath(routes.VolumeRemove, newVolumeScopePath, lookup, client),

		// Network routes
		newObjectListRoute(routes.NetworkList, lookup),
		newObjectInspectRoute(routes.NetworkInspect, newNetworkScopePath, lookup, client),
		newNetworkCreateRoute(lookup),
		newScopeObjectPath(routes.NetworkRemove, newNetworkScopePath, lookup, client),
		newScopeObjectPath(routes.NetworkConnect, newNetworkScopePath, lookup, client),
		newNetworkConnectRoute(routes.NetworkConnect, lookup, client),
		newScopeObjectPath(routes.NetworkDisconnect, newNetworkScopePath, lookup, client),
		newNetworkConnectRoute(routes.NetworkDisconnect, lookup, client),

		// Service routes
		newServiceListRoute(lookup),
		newServiceCreateRoute(lookup),
		newServiceInspectRoute(lookup, client),
		newScopeObjectPath(routes.ServiceRemove, newServiceScopePath, lookup, client),
		newScopeObjectPath(routes.ServiceUpdate, newServiceScopePath, lookup, client),
		newObjectUpdateRoute(routes.ServiceUpdate, lookup),
		newScopeObjectPath(routes.ServiceLogs, newServiceScopePath, lookup, client),

		// Secret routes
		newSecretInspectRoute(lookup, client),
		newSecretCreateRoute(lookup),
		newSecretListRoute(lookup),
		newScopeObjectPath(routes.SecretRemove, newSecretScopePath, lookup, client),
		newScopeObjectPath(routes.SecretUpdate, newSecretScopePath, lookup, client),
		newObjectUpdateRoute(routes.SecretUpdate, lookup),
	}
}
