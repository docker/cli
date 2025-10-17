package image

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	imageTagFunc     func(string, string) error
	imageSaveFunc    func(images []string, options ...client.ImageSaveOption) (io.ReadCloser, error)
	imageRemoveFunc  func(image string, options client.ImageRemoveOptions) ([]image.DeleteResponse, error)
	imagePushFunc    func(ref string, options client.ImagePushOptions) (io.ReadCloser, error)
	infoFunc         func() (system.Info, error)
	imagePullFunc    func(ref string, options client.ImagePullOptions) (client.ImagePullResponse, error)
	imagesPruneFunc  func(options client.ImagePruneOptions) (client.ImagePruneResult, error)
	imageLoadFunc    func(input io.Reader, options ...client.ImageLoadOption) (client.LoadResponse, error)
	imageListFunc    func(options client.ImageListOptions) ([]image.Summary, error)
	imageInspectFunc func(img string) (image.InspectResponse, error)
	imageImportFunc  func(source client.ImageImportSource, ref string, options client.ImageImportOptions) (io.ReadCloser, error)
	imageHistoryFunc func(img string, options ...client.ImageHistoryOption) ([]image.HistoryResponseItem, error)
	imageBuildFunc   func(context.Context, io.Reader, client.ImageBuildOptions) (client.ImageBuildResponse, error)
}

func (cli *fakeClient) ImageTag(_ context.Context, img, ref string) error {
	if cli.imageTagFunc != nil {
		return cli.imageTagFunc(img, ref)
	}
	return nil
}

func (cli *fakeClient) ImageSave(_ context.Context, images []string, options ...client.ImageSaveOption) (io.ReadCloser, error) {
	if cli.imageSaveFunc != nil {
		return cli.imageSaveFunc(images, options...)
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (cli *fakeClient) ImageRemove(_ context.Context, img string,
	options client.ImageRemoveOptions,
) ([]image.DeleteResponse, error) {
	if cli.imageRemoveFunc != nil {
		return cli.imageRemoveFunc(img, options)
	}
	return []image.DeleteResponse{}, nil
}

func (cli *fakeClient) ImagePush(_ context.Context, ref string, options client.ImagePushOptions) (io.ReadCloser, error) {
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

func (cli *fakeClient) ImagePull(_ context.Context, ref string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
	if cli.imagePullFunc != nil {
		return cli.imagePullFunc(ref, options)
	}
	return client.ImagePullResponse{}, nil
}

func (cli *fakeClient) ImagesPrune(_ context.Context, opts client.ImagePruneOptions) (client.ImagePruneResult, error) {
	if cli.imagesPruneFunc != nil {
		return cli.imagesPruneFunc(opts)
	}
	return client.ImagePruneResult{}, nil
}

func (cli *fakeClient) ImageLoad(_ context.Context, input io.Reader, options ...client.ImageLoadOption) (client.LoadResponse, error) {
	if cli.imageLoadFunc != nil {
		return cli.imageLoadFunc(input, options...)
	}
	return client.LoadResponse{}, nil
}

func (cli *fakeClient) ImageList(_ context.Context, options client.ImageListOptions) ([]image.Summary, error) {
	if cli.imageListFunc != nil {
		return cli.imageListFunc(options)
	}
	return []image.Summary{}, nil
}

func (cli *fakeClient) ImageInspect(_ context.Context, img string, _ ...client.ImageInspectOption) (image.InspectResponse, error) {
	if cli.imageInspectFunc != nil {
		return cli.imageInspectFunc(img)
	}
	return image.InspectResponse{}, nil
}

func (cli *fakeClient) ImageImport(_ context.Context, source client.ImageImportSource, ref string,
	options client.ImageImportOptions,
) (io.ReadCloser, error) {
	if cli.imageImportFunc != nil {
		return cli.imageImportFunc(source, ref, options)
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (cli *fakeClient) ImageHistory(_ context.Context, img string, options ...client.ImageHistoryOption) ([]image.HistoryResponseItem, error) {
	if cli.imageHistoryFunc != nil {
		return cli.imageHistoryFunc(img, options...)
	}
	return []image.HistoryResponseItem{{ID: img, Created: time.Now().Unix()}}, nil
}

func (cli *fakeClient) ImageBuild(ctx context.Context, buildContext io.Reader, options client.ImageBuildOptions) (client.ImageBuildResponse, error) {
	if cli.imageBuildFunc != nil {
		return cli.imageBuildFunc(ctx, buildContext, options)
	}
	return client.ImageBuildResponse{Body: io.NopCloser(strings.NewReader(""))}, nil
}
