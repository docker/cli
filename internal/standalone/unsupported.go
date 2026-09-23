package standalone

import (
	"context"
	"fmt"
	"io"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"
)

// unsupported provides an implementation of every [client.APIClient] method
// that returns a "not implemented" error. The standalone client embeds it and
// overrides the methods that it supports, so that the full interface is always
// satisfied even when new methods are added to the client.
type unsupported struct{}

func errNotSupported(op string) error {
	return fmt.Errorf("%w: %s is not supported in standalone mode (DOCKER_STANDALONE)", cerrdefs.ErrNotImplemented, op)
}

func (unsupported) CheckpointCreate(_ context.Context, _ string, _ client.CheckpointCreateOptions) (client.CheckpointCreateResult, error) {
	return client.CheckpointCreateResult{}, errNotSupported("CheckpointCreate")
}

func (unsupported) CheckpointRemove(_ context.Context, _ string, _ client.CheckpointRemoveOptions) (client.CheckpointRemoveResult, error) {
	return client.CheckpointRemoveResult{}, errNotSupported("CheckpointRemove")
}

func (unsupported) CheckpointList(_ context.Context, _ string, _ client.CheckpointListOptions) (client.CheckpointListResult, error) {
	return client.CheckpointListResult{}, errNotSupported("CheckpointList")
}

func (unsupported) ContainerCreate(_ context.Context, _ client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	return client.ContainerCreateResult{}, errNotSupported("ContainerCreate")
}

func (unsupported) ContainerInspect(_ context.Context, _ string, _ client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{}, errNotSupported("ContainerInspect")
}

func (unsupported) ContainerList(_ context.Context, _ client.ContainerListOptions) (client.ContainerListResult, error) {
	return client.ContainerListResult{}, errNotSupported("ContainerList")
}

func (unsupported) ContainerUpdate(_ context.Context, _ string, _ client.ContainerUpdateOptions) (client.ContainerUpdateResult, error) {
	return client.ContainerUpdateResult{}, errNotSupported("ContainerUpdate")
}

func (unsupported) ContainerRemove(_ context.Context, _ string, _ client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	return client.ContainerRemoveResult{}, errNotSupported("ContainerRemove")
}

func (unsupported) ContainerPrune(_ context.Context, _ client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
	return client.ContainerPruneResult{}, errNotSupported("ContainerPrune")
}

func (unsupported) ContainerLogs(_ context.Context, _ string, _ client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
	return nil, errNotSupported("ContainerLogs")
}

func (unsupported) ContainerStart(_ context.Context, _ string, _ client.ContainerStartOptions) (client.ContainerStartResult, error) {
	return client.ContainerStartResult{}, errNotSupported("ContainerStart")
}

func (unsupported) ContainerStop(_ context.Context, _ string, _ client.ContainerStopOptions) (client.ContainerStopResult, error) {
	return client.ContainerStopResult{}, errNotSupported("ContainerStop")
}

func (unsupported) ContainerRestart(_ context.Context, _ string, _ client.ContainerRestartOptions) (client.ContainerRestartResult, error) {
	return client.ContainerRestartResult{}, errNotSupported("ContainerRestart")
}

func (unsupported) ContainerPause(_ context.Context, _ string, _ client.ContainerPauseOptions) (client.ContainerPauseResult, error) {
	return client.ContainerPauseResult{}, errNotSupported("ContainerPause")
}

func (unsupported) ContainerUnpause(_ context.Context, _ string, _ client.ContainerUnpauseOptions) (client.ContainerUnpauseResult, error) {
	return client.ContainerUnpauseResult{}, errNotSupported("ContainerUnpause")
}

func (unsupported) ContainerKill(_ context.Context, _ string, _ client.ContainerKillOptions) (client.ContainerKillResult, error) {
	return client.ContainerKillResult{}, errNotSupported("ContainerKill")
}

func (unsupported) ContainerRename(_ context.Context, _ string, _ client.ContainerRenameOptions) (client.ContainerRenameResult, error) {
	return client.ContainerRenameResult{}, errNotSupported("ContainerRename")
}

func (unsupported) ContainerResize(_ context.Context, _ string, _ client.ContainerResizeOptions) (client.ContainerResizeResult, error) {
	return client.ContainerResizeResult{}, errNotSupported("ContainerResize")
}

func (unsupported) ContainerAttach(_ context.Context, _ string, _ client.ContainerAttachOptions) (client.ContainerAttachResult, error) {
	return client.ContainerAttachResult{}, errNotSupported("ContainerAttach")
}

func (unsupported) ContainerCommit(_ context.Context, _ string, _ client.ContainerCommitOptions) (client.ContainerCommitResult, error) {
	return client.ContainerCommitResult{}, errNotSupported("ContainerCommit")
}

func (unsupported) ContainerDiff(_ context.Context, _ string, _ client.ContainerDiffOptions) (client.ContainerDiffResult, error) {
	return client.ContainerDiffResult{}, errNotSupported("ContainerDiff")
}

func (unsupported) ContainerExport(_ context.Context, _ string, _ client.ContainerExportOptions) (client.ContainerExportResult, error) {
	return nil, errNotSupported("ContainerExport")
}

func (unsupported) ContainerStats(_ context.Context, _ string, _ client.ContainerStatsOptions) (client.ContainerStatsResult, error) {
	return client.ContainerStatsResult{}, errNotSupported("ContainerStats")
}

func (unsupported) ContainerTop(_ context.Context, _ string, _ client.ContainerTopOptions) (client.ContainerTopResult, error) {
	return client.ContainerTopResult{}, errNotSupported("ContainerTop")
}

func (unsupported) ContainerStatPath(_ context.Context, _ string, _ client.ContainerStatPathOptions) (client.ContainerStatPathResult, error) {
	return client.ContainerStatPathResult{}, errNotSupported("ContainerStatPath")
}

func (unsupported) CopyFromContainer(_ context.Context, _ string, _ client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
	return client.CopyFromContainerResult{}, errNotSupported("CopyFromContainer")
}

func (unsupported) CopyToContainer(_ context.Context, _ string, _ client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
	return client.CopyToContainerResult{}, errNotSupported("CopyToContainer")
}

func (unsupported) ExecCreate(_ context.Context, _ string, _ client.ExecCreateOptions) (client.ExecCreateResult, error) {
	return client.ExecCreateResult{}, errNotSupported("ExecCreate")
}

func (unsupported) ExecInspect(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
	return client.ExecInspectResult{}, errNotSupported("ExecInspect")
}

func (unsupported) ExecResize(_ context.Context, _ string, _ client.ExecResizeOptions) (client.ExecResizeResult, error) {
	return client.ExecResizeResult{}, errNotSupported("ExecResize")
}

func (unsupported) ExecStart(_ context.Context, _ string, _ client.ExecStartOptions) (client.ExecStartResult, error) {
	return client.ExecStartResult{}, errNotSupported("ExecStart")
}

func (unsupported) ExecAttach(_ context.Context, _ string, _ client.ExecAttachOptions) (client.ExecAttachResult, error) {
	return client.ExecAttachResult{}, errNotSupported("ExecAttach")
}

func (unsupported) DistributionInspect(_ context.Context, _ string, _ client.DistributionInspectOptions) (client.DistributionInspectResult, error) {
	return client.DistributionInspectResult{}, errNotSupported("DistributionInspect")
}

func (unsupported) ImageSearch(_ context.Context, _ string, _ client.ImageSearchOptions) (client.ImageSearchResult, error) {
	return client.ImageSearchResult{}, errNotSupported("ImageSearch")
}

func (unsupported) ImageBuild(_ context.Context, _ io.Reader, _ client.ImageBuildOptions) (client.ImageBuildResult, error) {
	return client.ImageBuildResult{}, errNotSupported("ImageBuild")
}

func (unsupported) BuildCachePrune(_ context.Context, _ client.BuildCachePruneOptions) (client.BuildCachePruneResult, error) {
	return client.BuildCachePruneResult{}, errNotSupported("BuildCachePrune")
}

func (unsupported) BuildCancel(_ context.Context, _ string, _ client.BuildCancelOptions) (client.BuildCancelResult, error) {
	return client.BuildCancelResult{}, errNotSupported("BuildCancel")
}

func (unsupported) ImageImport(_ context.Context, _ client.ImageImportSource, _ string, _ client.ImageImportOptions) (client.ImageImportResult, error) {
	return nil, errNotSupported("ImageImport")
}

func (unsupported) ImageList(_ context.Context, _ client.ImageListOptions) (client.ImageListResult, error) {
	return client.ImageListResult{}, errNotSupported("ImageList")
}

func (unsupported) ImagePull(_ context.Context, _ string, _ client.ImagePullOptions) (client.ImagePullResponse, error) {
	return nil, errNotSupported("ImagePull")
}

func (unsupported) ImagePush(_ context.Context, _ string, _ client.ImagePushOptions) (client.ImagePushResponse, error) {
	return nil, errNotSupported("ImagePush")
}

func (unsupported) ImageRemove(_ context.Context, _ string, _ client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
	return client.ImageRemoveResult{}, errNotSupported("ImageRemove")
}

func (unsupported) ImageTag(_ context.Context, _ client.ImageTagOptions) (client.ImageTagResult, error) {
	return client.ImageTagResult{}, errNotSupported("ImageTag")
}

func (unsupported) ImagePrune(_ context.Context, _ client.ImagePruneOptions) (client.ImagePruneResult, error) {
	return client.ImagePruneResult{}, errNotSupported("ImagePrune")
}

func (unsupported) ImageInspect(_ context.Context, _ string, _ ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	return client.ImageInspectResult{}, errNotSupported("ImageInspect")
}

func (unsupported) ImageHistory(_ context.Context, _ string, _ ...client.ImageHistoryOption) (client.ImageHistoryResult, error) {
	return client.ImageHistoryResult{}, errNotSupported("ImageHistory")
}

func (unsupported) ImageAttestations(_ context.Context, _ string, _ ...client.ImageAttestationsOption) (client.ImageAttestationsResult, error) {
	return client.ImageAttestationsResult{}, errNotSupported("ImageAttestations")
}

func (unsupported) ImageLoad(_ context.Context, _ io.Reader, _ ...client.ImageLoadOption) (client.ImageLoadResult, error) {
	return nil, errNotSupported("ImageLoad")
}

func (unsupported) ImageSave(_ context.Context, _ []string, _ ...client.ImageSaveOption) (client.ImageSaveResult, error) {
	return nil, errNotSupported("ImageSave")
}

func (unsupported) NetworkCreate(_ context.Context, _ string, _ client.NetworkCreateOptions) (client.NetworkCreateResult, error) {
	return client.NetworkCreateResult{}, errNotSupported("NetworkCreate")
}

func (unsupported) NetworkInspect(_ context.Context, _ string, _ client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
	return client.NetworkInspectResult{}, errNotSupported("NetworkInspect")
}

func (unsupported) NetworkList(_ context.Context, _ client.NetworkListOptions) (client.NetworkListResult, error) {
	return client.NetworkListResult{}, errNotSupported("NetworkList")
}

func (unsupported) NetworkRemove(_ context.Context, _ string, _ client.NetworkRemoveOptions) (client.NetworkRemoveResult, error) {
	return client.NetworkRemoveResult{}, errNotSupported("NetworkRemove")
}

func (unsupported) NetworkPrune(_ context.Context, _ client.NetworkPruneOptions) (client.NetworkPruneResult, error) {
	return client.NetworkPruneResult{}, errNotSupported("NetworkPrune")
}

func (unsupported) NetworkConnect(_ context.Context, _ string, _ client.NetworkConnectOptions) (client.NetworkConnectResult, error) {
	return client.NetworkConnectResult{}, errNotSupported("NetworkConnect")
}

func (unsupported) NetworkDisconnect(_ context.Context, _ string, _ client.NetworkDisconnectOptions) (client.NetworkDisconnectResult, error) {
	return client.NetworkDisconnectResult{}, errNotSupported("NetworkDisconnect")
}

func (unsupported) NodeInspect(_ context.Context, _ string, _ client.NodeInspectOptions) (client.NodeInspectResult, error) {
	return client.NodeInspectResult{}, errNotSupported("NodeInspect")
}

func (unsupported) NodeList(_ context.Context, _ client.NodeListOptions) (client.NodeListResult, error) {
	return client.NodeListResult{}, errNotSupported("NodeList")
}

func (unsupported) NodeUpdate(_ context.Context, _ string, _ client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
	return client.NodeUpdateResult{}, errNotSupported("NodeUpdate")
}

func (unsupported) NodeRemove(_ context.Context, _ string, _ client.NodeRemoveOptions) (client.NodeRemoveResult, error) {
	return client.NodeRemoveResult{}, errNotSupported("NodeRemove")
}

func (unsupported) PluginCreate(_ context.Context, _ io.Reader, _ client.PluginCreateOptions) (client.PluginCreateResult, error) {
	return client.PluginCreateResult{}, errNotSupported("PluginCreate")
}

func (unsupported) PluginInstall(_ context.Context, _ string, _ client.PluginInstallOptions) (client.PluginInstallResult, error) {
	return client.PluginInstallResult{}, errNotSupported("PluginInstall")
}

func (unsupported) PluginInspect(_ context.Context, _ string, _ client.PluginInspectOptions) (client.PluginInspectResult, error) {
	return client.PluginInspectResult{}, errNotSupported("PluginInspect")
}

func (unsupported) PluginList(_ context.Context, _ client.PluginListOptions) (client.PluginListResult, error) {
	return client.PluginListResult{}, errNotSupported("PluginList")
}

func (unsupported) PluginRemove(_ context.Context, _ string, _ client.PluginRemoveOptions) (client.PluginRemoveResult, error) {
	return client.PluginRemoveResult{}, errNotSupported("PluginRemove")
}

func (unsupported) PluginEnable(_ context.Context, _ string, _ client.PluginEnableOptions) (client.PluginEnableResult, error) {
	return client.PluginEnableResult{}, errNotSupported("PluginEnable")
}

func (unsupported) PluginDisable(_ context.Context, _ string, _ client.PluginDisableOptions) (client.PluginDisableResult, error) {
	return client.PluginDisableResult{}, errNotSupported("PluginDisable")
}

func (unsupported) PluginUpgrade(_ context.Context, _ string, _ client.PluginUpgradeOptions) (client.PluginUpgradeResult, error) {
	return nil, errNotSupported("PluginUpgrade")
}

func (unsupported) PluginPush(_ context.Context, _ string, _ client.PluginPushOptions) (client.PluginPushResult, error) {
	return client.PluginPushResult{}, errNotSupported("PluginPush")
}

func (unsupported) PluginSet(_ context.Context, _ string, _ client.PluginSetOptions) (client.PluginSetResult, error) {
	return client.PluginSetResult{}, errNotSupported("PluginSet")
}

func (unsupported) ServiceCreate(_ context.Context, _ client.ServiceCreateOptions) (client.ServiceCreateResult, error) {
	return client.ServiceCreateResult{}, errNotSupported("ServiceCreate")
}

func (unsupported) ServiceInspect(_ context.Context, _ string, _ client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	return client.ServiceInspectResult{}, errNotSupported("ServiceInspect")
}

func (unsupported) ServiceList(_ context.Context, _ client.ServiceListOptions) (client.ServiceListResult, error) {
	return client.ServiceListResult{}, errNotSupported("ServiceList")
}

func (unsupported) ServiceUpdate(_ context.Context, _ string, _ client.ServiceUpdateOptions) (client.ServiceUpdateResult, error) {
	return client.ServiceUpdateResult{}, errNotSupported("ServiceUpdate")
}

func (unsupported) ServiceRemove(_ context.Context, _ string, _ client.ServiceRemoveOptions) (client.ServiceRemoveResult, error) {
	return client.ServiceRemoveResult{}, errNotSupported("ServiceRemove")
}

func (unsupported) ServiceLogs(_ context.Context, _ string, _ client.ServiceLogsOptions) (client.ServiceLogsResult, error) {
	return nil, errNotSupported("ServiceLogs")
}

func (unsupported) TaskInspect(_ context.Context, _ string, _ client.TaskInspectOptions) (client.TaskInspectResult, error) {
	return client.TaskInspectResult{}, errNotSupported("TaskInspect")
}

func (unsupported) TaskList(_ context.Context, _ client.TaskListOptions) (client.TaskListResult, error) {
	return client.TaskListResult{}, errNotSupported("TaskList")
}

func (unsupported) TaskLogs(_ context.Context, _ string, _ client.TaskLogsOptions) (client.TaskLogsResult, error) {
	return nil, errNotSupported("TaskLogs")
}

func (unsupported) SwarmInit(_ context.Context, _ client.SwarmInitOptions) (client.SwarmInitResult, error) {
	return client.SwarmInitResult{}, errNotSupported("SwarmInit")
}

func (unsupported) SwarmJoin(_ context.Context, _ client.SwarmJoinOptions) (client.SwarmJoinResult, error) {
	return client.SwarmJoinResult{}, errNotSupported("SwarmJoin")
}

func (unsupported) SwarmInspect(_ context.Context, _ client.SwarmInspectOptions) (client.SwarmInspectResult, error) {
	return client.SwarmInspectResult{}, errNotSupported("SwarmInspect")
}

func (unsupported) SwarmUpdate(_ context.Context, _ client.SwarmUpdateOptions) (client.SwarmUpdateResult, error) {
	return client.SwarmUpdateResult{}, errNotSupported("SwarmUpdate")
}

func (unsupported) SwarmLeave(_ context.Context, _ client.SwarmLeaveOptions) (client.SwarmLeaveResult, error) {
	return client.SwarmLeaveResult{}, errNotSupported("SwarmLeave")
}

func (unsupported) SwarmGetUnlockKey(_ context.Context) (client.SwarmGetUnlockKeyResult, error) {
	return client.SwarmGetUnlockKeyResult{}, errNotSupported("SwarmGetUnlockKey")
}

func (unsupported) SwarmUnlock(_ context.Context, _ client.SwarmUnlockOptions) (client.SwarmUnlockResult, error) {
	return client.SwarmUnlockResult{}, errNotSupported("SwarmUnlock")
}

func (unsupported) RegistryLogin(_ context.Context, _ client.RegistryLoginOptions) (client.RegistryLoginResult, error) {
	return client.RegistryLoginResult{}, errNotSupported("RegistryLogin")
}

func (unsupported) DiskUsage(_ context.Context, _ client.DiskUsageOptions) (client.DiskUsageResult, error) {
	return client.DiskUsageResult{}, errNotSupported("DiskUsage")
}

func (unsupported) VolumeCreate(_ context.Context, _ client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
	return client.VolumeCreateResult{}, errNotSupported("VolumeCreate")
}

func (unsupported) VolumeInspect(_ context.Context, _ string, _ client.VolumeInspectOptions) (client.VolumeInspectResult, error) {
	return client.VolumeInspectResult{}, errNotSupported("VolumeInspect")
}

func (unsupported) VolumeList(_ context.Context, _ client.VolumeListOptions) (client.VolumeListResult, error) {
	return client.VolumeListResult{}, errNotSupported("VolumeList")
}

func (unsupported) VolumeUpdate(_ context.Context, _ string, _ client.VolumeUpdateOptions) (client.VolumeUpdateResult, error) {
	return client.VolumeUpdateResult{}, errNotSupported("VolumeUpdate")
}

func (unsupported) VolumeRemove(_ context.Context, _ string, _ client.VolumeRemoveOptions) (client.VolumeRemoveResult, error) {
	return client.VolumeRemoveResult{}, errNotSupported("VolumeRemove")
}

func (unsupported) VolumePrune(_ context.Context, _ client.VolumePruneOptions) (client.VolumePruneResult, error) {
	return client.VolumePruneResult{}, errNotSupported("VolumePrune")
}

func (unsupported) SecretCreate(_ context.Context, _ client.SecretCreateOptions) (client.SecretCreateResult, error) {
	return client.SecretCreateResult{}, errNotSupported("SecretCreate")
}

func (unsupported) SecretInspect(_ context.Context, _ string, _ client.SecretInspectOptions) (client.SecretInspectResult, error) {
	return client.SecretInspectResult{}, errNotSupported("SecretInspect")
}

func (unsupported) SecretList(_ context.Context, _ client.SecretListOptions) (client.SecretListResult, error) {
	return client.SecretListResult{}, errNotSupported("SecretList")
}

func (unsupported) SecretUpdate(_ context.Context, _ string, _ client.SecretUpdateOptions) (client.SecretUpdateResult, error) {
	return client.SecretUpdateResult{}, errNotSupported("SecretUpdate")
}

func (unsupported) SecretRemove(_ context.Context, _ string, _ client.SecretRemoveOptions) (client.SecretRemoveResult, error) {
	return client.SecretRemoveResult{}, errNotSupported("SecretRemove")
}

func (unsupported) ConfigCreate(_ context.Context, _ client.ConfigCreateOptions) (client.ConfigCreateResult, error) {
	return client.ConfigCreateResult{}, errNotSupported("ConfigCreate")
}

func (unsupported) ConfigInspect(_ context.Context, _ string, _ client.ConfigInspectOptions) (client.ConfigInspectResult, error) {
	return client.ConfigInspectResult{}, errNotSupported("ConfigInspect")
}

func (unsupported) ConfigList(_ context.Context, _ client.ConfigListOptions) (client.ConfigListResult, error) {
	return client.ConfigListResult{}, errNotSupported("ConfigList")
}

func (unsupported) ConfigUpdate(_ context.Context, _ string, _ client.ConfigUpdateOptions) (client.ConfigUpdateResult, error) {
	return client.ConfigUpdateResult{}, errNotSupported("ConfigUpdate")
}

func (unsupported) ConfigRemove(_ context.Context, _ string, _ client.ConfigRemoveOptions) (client.ConfigRemoveResult, error) {
	return client.ConfigRemoveResult{}, errNotSupported("ConfigRemove")
}
