package image

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	imageTagFunc     func(string, string) error
	imageSaveFunc    func(images []string) (io.ReadCloser, error)
	imageRemoveFunc  func(image string, options image.RemoveOptions) ([]image.DeleteResponse, error)
	imagePushFunc    func(ref string, options image.PushOptions) (io.ReadCloser, error)
	infoFunc         func() (system.Info, error)
	imagePullFunc    func(ref string, options image.PullOptions) (io.ReadCloser, error)
	imagesPruneFunc  func(pruneFilter filters.Args) (types.ImagesPruneReport, error)
	imageLoadFunc    func(input io.Reader, quiet bool) (types.ImageLoadResponse, error)
	imageListFunc    func(options image.ListOptions) ([]image.Summary, error)
	imageInspectFunc func(image string) (types.ImageInspect, []byte, error)
	imageImportFunc  func(source types.ImageImportSource, ref string, options image.ImportOptions) (io.ReadCloser, error)
	imageHistoryFunc func(image string) ([]image.HistoryResponseItem, error)
	imageBuildFunc   func(context.Context, io.Reader, types.ImageBuildOptions) (types.ImageBuildResponse, error)
}

func (cli *fakeClient) ImageTag(_ context.Context, img, ref string) error {
	if cli.imageTagFunc != nil {
		return cli.imageTagFunc(img, ref)
	}
	return nil
}

func (cli *fakeClient) ImageSave(_ context.Context, images []string) (io.ReadCloser, error) {
	if cli.imageSaveFunc != nil {
		return cli.imageSaveFunc(images)
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (cli *fakeClient) ImageRemove(_ context.Context, img string,
	options image.RemoveOptions,
) ([]image.DeleteResponse, error) {
	if cli.imageRemoveFunc != nil {
		return cli.imageRemoveFunc(img, options)
	}
	return []image.DeleteResponse{}, nil
}

func (cli *fakeClient) ImagePush(_ context.Context, ref string, options image.PushOptions) (io.ReadCloser, error) {
	if cli.imagePushFunc != nil {
		return cli.imagePushFunc(ref, options)
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (cli *fakeClient) Info(_ context.Context) (system.Info, error) {
	if cli.infoFunc != nil {
		return cli.infoFunc()
	}
	return system.Info{}, nil
}

func (cli *fakeClient) ImagePull(_ context.Context, ref string, options image.PullOptions) (io.ReadCloser, error) {
	if cli.imagePullFunc != nil {
		cli.imagePullFunc(ref, options)
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (cli *fakeClient) ImagesPrune(_ context.Context, pruneFilter filters.Args) (types.ImagesPruneReport, error) {
	if cli.imagesPruneFunc != nil {
		return cli.imagesPruneFunc(pruneFilter)
	}
	return types.ImagesPruneReport{}, nil
}

func (cli *fakeClient) ImageLoad(_ context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
	if cli.imageLoadFunc != nil {
		return cli.imageLoadFunc(input, quiet)
	}
	return types.ImageLoadResponse{}, nil
}

func (cli *fakeClient) ImageList(_ context.Context, options image.ListOptions) ([]image.Summary, error) {
	if cli.imageListFunc != nil {
		return cli.imageListFunc(options)
	}
	return []image.Summary{}, nil
}

func (cli *fakeClient) ImageInspectWithRaw(_ context.Context, img string) (types.ImageInspect, []byte, error) {
	if cli.imageInspectFunc != nil {
		return cli.imageInspectFunc(img)
	}
	return types.ImageInspect{}, nil, nil
}

func (cli *fakeClient) ImageImport(_ context.Context, source types.ImageImportSource, ref string,
	options image.ImportOptions,
) (io.ReadCloser, error) {
	if cli.imageImportFunc != nil {
		return cli.imageImportFunc(source, ref, options)
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (cli *fakeClient) ImageHistory(_ context.Context, img string) ([]image.HistoryResponseItem, error) {
	if cli.imageHistoryFunc != nil {
		return cli.imageHistoryFunc(img)
	}
	return []image.HistoryResponseItem{{ID: img, Created: time.Now().Unix()}}, nil
}

func (cli *fakeClient) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	if cli.imageBuildFunc != nil {
		return cli.imageBuildFunc(ctx, buildContext, options)
	}
	return types.ImageBuildResponse{Body: io.NopCloser(strings.NewReader(""))}, nil
}
