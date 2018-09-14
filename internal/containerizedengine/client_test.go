package containerizedengine

import (
	"context"
	"syscall"

	"github.com/containerd/containerd"
	containerdtypes "github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/oci"
	prototypes "github.com/gogo/protobuf/types"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type (
	fakeContainerdClient struct {
		containersFunc       func(ctx context.Context, filters ...string) ([]containerd.Container, error)
		newContainerFunc     func(ctx context.Context, id string, opts ...containerd.NewContainerOpts) (containerd.Container, error)
		pullFunc             func(ctx context.Context, ref string, opts ...containerd.RemoteOpt) (containerd.Image, error)
		getImageFunc         func(ctx context.Context, ref string) (containerd.Image, error)
		contentStoreFunc     func() content.Store
		containerServiceFunc func() containers.Store
		installFunc          func(context.Context, containerd.Image, ...containerd.InstallOpts) error
		versionFunc          func(ctx context.Context) (containerd.Version, error)
	}
	fakeContainer struct {
		idFunc         func() string
		infoFunc       func(context.Context) (containers.Container, error)
		deleteFunc     func(context.Context, ...containerd.DeleteOpts) error
		newTaskFunc    func(context.Context, cio.Creator, ...containerd.NewTaskOpts) (containerd.Task, error)
		specFunc       func(context.Context) (*oci.Spec, error)
		taskFunc       func(context.Context, cio.Attach) (containerd.Task, error)
		imageFunc      func(context.Context) (containerd.Image, error)
		labelsFunc     func(context.Context) (map[string]string, error)
		setLabelsFunc  func(context.Context, map[string]string) (map[string]string, error)
		extensionsFunc func(context.Context) (map[string]prototypes.Any, error)
		updateFunc     func(context.Context, ...containerd.UpdateContainerOpts) error
	}
	fakeImage struct {
		nameFunc         func() string
		targetFunc       func() ocispec.Descriptor
		unpackFunc       func(context.Context, string) error
		rootFSFunc       func(ctx context.Context) ([]digest.Digest, error)
		sizeFunc         func(ctx context.Context) (int64, error)
		configFunc       func(ctx context.Context) (ocispec.Descriptor, error)
		isUnpackedFunc   func(context.Context, string) (bool, error)
		contentStoreFunc func() content.Store
	}
	fakeTask struct {
		idFunc          func() string
		pidFunc         func() uint32
		startFunc       func(context.Context) error
		deleteFunc      func(context.Context, ...containerd.ProcessDeleteOpts) (*containerd.ExitStatus, error)
		killFunc        func(context.Context, syscall.Signal, ...containerd.KillOpts) error
		waitFunc        func(context.Context) (<-chan containerd.ExitStatus, error)
		closeIOFunc     func(context.Context, ...containerd.IOCloserOpts) error
		resizeFunc      func(ctx context.Context, w, h uint32) error
		ioFunc          func() cio.IO
		statusFunc      func(context.Context) (containerd.Status, error)
		pauseFunc       func(context.Context) error
		resumeFunc      func(context.Context) error
		execFunc        func(context.Context, string, *specs.Process, cio.Creator) (containerd.Process, error)
		pidsFunc        func(context.Context) ([]containerd.ProcessInfo, error)
		checkpointFunc  func(context.Context, ...containerd.CheckpointTaskOpts) (containerd.Image, error)
		updateFunc      func(context.Context, ...containerd.UpdateTaskOpts) error
		loadProcessFunc func(context.Context, string, cio.Attach) (containerd.Process, error)
		metricsFunc     func(context.Context) (*containerdtypes.Metric, error)
	}
)

func (w *fakeContainerdClient) Containers(ctx context.Context, filters ...string) ([]containerd.Container, error) {
	if w.containersFunc != nil {
		return w.containersFunc(ctx, filters...)
	}
	return []containerd.Container{}, nil
}
func (w *fakeContainerdClient) NewContainer(ctx context.Context, id string, opts ...containerd.NewContainerOpts) (containerd.Container, error) {
	if w.newContainerFunc != nil {
		return w.newContainerFunc(ctx, id, opts...)
	}
	return nil, nil
}
func (w *fakeContainerdClient) Pull(ctx context.Context, ref string, opts ...containerd.RemoteOpt) (containerd.Image, error) {
	if w.pullFunc != nil {
		return w.pullFunc(ctx, ref, opts...)
	}
	return nil, nil
}
func (w *fakeContainerdClient) GetImage(ctx context.Context, ref string) (containerd.Image, error) {
	if w.getImageFunc != nil {
		return w.getImageFunc(ctx, ref)
	}
	return nil, nil
}
func (w *fakeContainerdClient) ContentStore() content.Store {
	if w.contentStoreFunc != nil {
		return w.contentStoreFunc()
	}
	return nil
}
func (w *fakeContainerdClient) ContainerService() containers.Store {
	if w.containerServiceFunc != nil {
		return w.containerServiceFunc()
	}
	return nil
}
func (w *fakeContainerdClient) Close() error {
	return nil
}
func (w *fakeContainerdClient) Install(ctx context.Context, image containerd.Image, args ...containerd.InstallOpts) error {
	if w.installFunc != nil {
		return w.installFunc(ctx, image, args...)
	}
	return nil
}
func (w *fakeContainerdClient) Version(ctx context.Context) (containerd.Version, error) {
	if w.versionFunc != nil {
		return w.versionFunc(ctx)
	}
	return containerd.Version{}, nil
}

func (c *fakeContainer) ID() string {
	if c.idFunc != nil {
		return c.idFunc()
	}
	return ""
}
func (c *fakeContainer) Info(ctx context.Context) (containers.Container, error) {
	if c.infoFunc != nil {
		return c.infoFunc(ctx)
	}
	return containers.Container{}, nil
}
func (c *fakeContainer) Delete(ctx context.Context, opts ...containerd.DeleteOpts) error {
	if c.deleteFunc != nil {
		return c.deleteFunc(ctx, opts...)
	}
	return nil
}
func (c *fakeContainer) NewTask(ctx context.Context, ioc cio.Creator, opts ...containerd.NewTaskOpts) (containerd.Task, error) {
	if c.newTaskFunc != nil {
		return c.newTaskFunc(ctx, ioc, opts...)
	}
	return nil, nil
}
func (c *fakeContainer) Spec(ctx context.Context) (*oci.Spec, error) {
	if c.specFunc != nil {
		return c.specFunc(ctx)
	}
	return nil, nil
}
func (c *fakeContainer) Task(ctx context.Context, attach cio.Attach) (containerd.Task, error) {
	if c.taskFunc != nil {
		return c.taskFunc(ctx, attach)
	}
	return nil, nil
}
func (c *fakeContainer) Image(ctx context.Context) (containerd.Image, error) {
	if c.imageFunc != nil {
		return c.imageFunc(ctx)
	}
	return nil, nil
}
func (c *fakeContainer) Labels(ctx context.Context) (map[string]string, error) {
	if c.labelsFunc != nil {
		return c.labelsFunc(ctx)
	}
	return nil, nil
}
func (c *fakeContainer) SetLabels(ctx context.Context, labels map[string]string) (map[string]string, error) {
	if c.setLabelsFunc != nil {
		return c.setLabelsFunc(ctx, labels)
	}
	return nil, nil
}
func (c *fakeContainer) Extensions(ctx context.Context) (map[string]prototypes.Any, error) {
	if c.extensionsFunc != nil {
		return c.extensionsFunc(ctx)
	}
	return nil, nil
}
func (c *fakeContainer) Update(ctx context.Context, opts ...containerd.UpdateContainerOpts) error {
	if c.updateFunc != nil {
		return c.updateFunc(ctx, opts...)
	}
	return nil
}

func (i *fakeImage) Name() string {
	if i.nameFunc != nil {
		return i.nameFunc()
	}
	return ""
}
func (i *fakeImage) Target() ocispec.Descriptor {
	if i.targetFunc != nil {
		return i.targetFunc()
	}
	return ocispec.Descriptor{}
}
func (i *fakeImage) Unpack(ctx context.Context, name string) error {
	if i.unpackFunc != nil {
		return i.unpackFunc(ctx, name)
	}
	return nil
}
func (i *fakeImage) RootFS(ctx context.Context) ([]digest.Digest, error) {
	if i.rootFSFunc != nil {
		return i.rootFSFunc(ctx)
	}
	return nil, nil
}
func (i *fakeImage) Size(ctx context.Context) (int64, error) {
	if i.sizeFunc != nil {
		return i.sizeFunc(ctx)
	}
	return 0, nil
}
func (i *fakeImage) Config(ctx context.Context) (ocispec.Descriptor, error) {
	if i.configFunc != nil {
		return i.configFunc(ctx)
	}
	return ocispec.Descriptor{}, nil
}
func (i *fakeImage) IsUnpacked(ctx context.Context, name string) (bool, error) {
	if i.isUnpackedFunc != nil {
		return i.isUnpackedFunc(ctx, name)
	}
	return false, nil
}
func (i *fakeImage) ContentStore() content.Store {
	if i.contentStoreFunc != nil {
		return i.contentStoreFunc()
	}
	return nil
}

func (t *fakeTask) ID() string {
	if t.idFunc != nil {
		return t.idFunc()
	}
	return ""
}
func (t *fakeTask) Pid() uint32 {
	if t.pidFunc != nil {
		return t.pidFunc()
	}
	return 0
}
func (t *fakeTask) Start(ctx context.Context) error {
	if t.startFunc != nil {
		return t.startFunc(ctx)
	}
	return nil
}
func (t *fakeTask) Delete(ctx context.Context, opts ...containerd.ProcessDeleteOpts) (*containerd.ExitStatus, error) {
	if t.deleteFunc != nil {
		return t.deleteFunc(ctx, opts...)
	}
	return nil, nil
}
func (t *fakeTask) Kill(ctx context.Context, signal syscall.Signal, opts ...containerd.KillOpts) error {
	if t.killFunc != nil {
		return t.killFunc(ctx, signal, opts...)
	}
	return nil
}
func (t *fakeTask) Wait(ctx context.Context) (<-chan containerd.ExitStatus, error) {
	if t.waitFunc != nil {
		return t.waitFunc(ctx)
	}
	return nil, nil
}
func (t *fakeTask) CloseIO(ctx context.Context, opts ...containerd.IOCloserOpts) error {
	if t.closeIOFunc != nil {
		return t.closeIOFunc(ctx, opts...)
	}
	return nil
}
func (t *fakeTask) Resize(ctx context.Context, w, h uint32) error {
	if t.resizeFunc != nil {
		return t.resizeFunc(ctx, w, h)
	}
	return nil
}
func (t *fakeTask) IO() cio.IO {
	if t.ioFunc != nil {
		return t.ioFunc()
	}
	return nil
}
func (t *fakeTask) Status(ctx context.Context) (containerd.Status, error) {
	if t.statusFunc != nil {
		return t.statusFunc(ctx)
	}
	return containerd.Status{}, nil
}
func (t *fakeTask) Pause(ctx context.Context) error {
	if t.pauseFunc != nil {
		return t.pauseFunc(ctx)
	}
	return nil
}
func (t *fakeTask) Resume(ctx context.Context) error {
	if t.resumeFunc != nil {
		return t.resumeFunc(ctx)
	}
	return nil
}
func (t *fakeTask) Exec(ctx context.Context, cmd string, proc *specs.Process, ioc cio.Creator) (containerd.Process, error) {
	if t.execFunc != nil {
		return t.execFunc(ctx, cmd, proc, ioc)
	}
	return nil, nil
}
func (t *fakeTask) Pids(ctx context.Context) ([]containerd.ProcessInfo, error) {
	if t.pidsFunc != nil {
		return t.pidsFunc(ctx)
	}
	return nil, nil
}
func (t *fakeTask) Checkpoint(ctx context.Context, opts ...containerd.CheckpointTaskOpts) (containerd.Image, error) {
	if t.checkpointFunc != nil {
		return t.checkpointFunc(ctx, opts...)
	}
	return nil, nil
}
func (t *fakeTask) Update(ctx context.Context, opts ...containerd.UpdateTaskOpts) error {
	if t.updateFunc != nil {
		return t.updateFunc(ctx, opts...)
	}
	return nil
}
func (t *fakeTask) LoadProcess(ctx context.Context, name string, attach cio.Attach) (containerd.Process, error) {
	if t.loadProcessFunc != nil {
		return t.loadProcessFunc(ctx, name, attach)
	}
	return nil, nil
}
func (t *fakeTask) Metrics(ctx context.Context) (*containerdtypes.Metric, error) {
	if t.metricsFunc != nil {
		return t.metricsFunc(ctx)
	}
	return nil, nil
}
