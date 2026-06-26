package image

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	imageTagFunc     func(options client.ImageTagOptions) (client.ImageTagResult, error)
	imageSaveFunc    func(images []string, options ...client.ImageSaveOption) (client.ImageSaveResult, error)
	imageRemoveFunc  func(image string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error)
	imagePushFunc    func(ref string, options client.ImagePushOptions) (client.ImagePushResponse, error)
	infoFunc         func() (client.SystemInfoResult, error)
	imagePullFunc    func(ref string, options client.ImagePullOptions) (client.ImagePullResponse, error)
	imagePruneFunc   func(options client.ImagePruneOptions) (client.ImagePruneResult, error)
	imageLoadFunc    func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error)
	imageListFunc    func(options client.ImageListOptions) (client.ImageListResult, error)
	imageInspectFunc func(img string) (client.ImageInspectResult, error)
	imageImportFunc  func(source client.ImageImportSource, ref string, options client.ImageImportOptions) (client.ImageImportResult, error)
	imageHistoryFunc func(img string, options ...client.ImageHistoryOption) (client.ImageHistoryResult, error)
	imageBuildFunc   func(context.Context, io.Reader, client.ImageBuildOptions) (client.ImageBuildResult, error)
}

type fakeStreamResult struct {
	io.ReadCloser
	client.ImagePushResponse // same interface as [client.ImagePullResponse]
}

func (e fakeStreamResult) Read(p []byte) (int, error) { return e.ReadCloser.Read(p) }
func (e fakeStreamResult) Close() error               { return e.ReadCloser.Close() }

func (cli *fakeClient) ImageTag(_ context.Context, options client.ImageTagOptions) (client.ImageTagResult, error) {
	if cli.imageTagFunc != nil {
		return cli.imageTagFunc(options)
	}
	return client.ImageTagResult{}, nil
}

func (cli *fakeClient) ImageSave(_ context.Context, images []string, options ...client.ImageSaveOption) (client.ImageSaveResult, error) {
	if cli.imageSaveFunc != nil {
		return cli.imageSaveFunc(images, options...)
	}
	return http.NoBody, nil
}

func (cli *fakeClient) ImageRemove(_ context.Context, img string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
	if cli.imageRemoveFunc != nil {
		return cli.imageRemoveFunc(img, options)
	}
	return client.ImageRemoveResult{}, nil
}

func (cli *fakeClient) ImagePush(_ context.Context, ref string, options client.ImagePushOptions) (client.ImagePushResponse, error) {
	if cli.imagePushFunc != nil {
		return cli.imagePushFunc(ref, options)
	}
	// FIXME(thaJeztah): how to mock this?
	return fakeStreamResult{ReadCloser: http.NoBody}, nil
}

func (cli *fakeClient) Info(_ context.Context, _ client.InfoOptions) (client.SystemInfoResult, error) {
	if cli.infoFunc != nil {
		return cli.infoFunc()
	}
	return client.SystemInfoResult{}, nil
}

func (cli *fakeClient) ImagePull(_ context.Context, ref string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
	if cli.imagePullFunc != nil {
		return cli.imagePullFunc(ref, options)
	}
	// FIXME(thaJeztah): how to mock this?
	return fakeStreamResult{ReadCloser: http.NoBody}, nil
}

func (cli *fakeClient) ImagePrune(_ context.Context, opts client.ImagePruneOptions) (client.ImagePruneResult, error) {
	if cli.imagePruneFunc != nil {
		return cli.imagePruneFunc(opts)
	}
	return client.ImagePruneResult{}, nil
}

func (cli *fakeClient) ImageLoad(_ context.Context, input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error) {
	if cli.imageLoadFunc != nil {
		return cli.imageLoadFunc(input, options...)
	}
	return http.NoBody, nil
}

func (cli *fakeClient) ImageList(_ context.Context, options client.ImageListOptions) (client.ImageListResult, error) {
	if cli.imageListFunc != nil {
		return cli.imageListFunc(options)
	}
	return client.ImageListResult{}, nil
}

func (cli *fakeClient) ImageInspect(_ context.Context, img string, _ ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	if cli.imageInspectFunc != nil {
		return cli.imageInspectFunc(img)
	}
	return client.ImageInspectResult{}, nil
}

func (cli *fakeClient) ImageImport(_ context.Context, source client.ImageImportSource, ref string, options client.ImageImportOptions) (client.ImageImportResult, error) {
	if cli.imageImportFunc != nil {
		return cli.imageImportFunc(source, ref, options)
	}
	return http.NoBody, nil
}

func (cli *fakeClient) ImageHistory(_ context.Context, img string, options ...client.ImageHistoryOption) (client.ImageHistoryResult, error) {
	if cli.imageHistoryFunc != nil {
		return cli.imageHistoryFunc(img, options...)
	}
	return client.ImageHistoryResult{
		Items: []image.HistoryResponseItem{{ID: img, Created: time.Now().Unix()}},
	}, nil
}

func (cli *fakeClient) ImageBuild(ctx context.Context, buildContext io.Reader, options client.ImageBuildOptions) (client.ImageBuildResult, error) {
	if cli.imageBuildFunc != nil {
		return cli.imageBuildFunc(ctx, buildContext, options)
	}
	return client.ImageBuildResult{Body: io.NopCloser(strings.NewReader(""))}, nil
}
