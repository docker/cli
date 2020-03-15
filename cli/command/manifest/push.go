package manifest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/manifest/types"
	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

type pushOpts struct {
	insecure bool
	purge    bool
	file     string
	target   string
}

type mountRequest struct {
	ref      reference.Named
	manifest types.ImageManifest
}

type manifestBlob struct {
	canonical reference.Canonical
	os        string
}

type pushRequest struct {
	targetRef     reference.Named
	list          *manifestlist.DeserializedManifestList
	mountRequests []mountRequest
	manifestBlobs []manifestBlob
	insecure      bool
}

type yamlManifestList struct {
	Image     string
	Manifests []yamlManifest
}

type yamlManifest struct {
	Image    string
	Platform manifestlist.PlatformSpec
}

func newPushListCommand(dockerCli command.Cli) *cobra.Command {
	opts := pushOpts{}

	cmd := &cobra.Command{
		Use:   "push [OPTIONS] MANIFEST_LIST",
		Short: "Push a manifest list to a repository, either after a create, or from a file",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.target = args[0]
			return runPush(dockerCli, opts)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.purge, "purge", "p", false, "remove the local manifests after push")
	flags.BoolVar(&opts.insecure, "insecure", false, "allow push to an insecure registry")
	flags.StringVar(&opts.file, "file", "", "file containing the yaml representation of manifest list")
	return cmd
}

func runPush(dockerCli command.Cli, opts pushOpts) error {
	targetRef, err := normalizeReference(opts.target)
	if err != nil {
		return err
	}
	if opts.file != "" {
		return pushListFromYaml(dockerCli, targetRef, opts)
	}

	return pushListFromStore(dockerCli, targetRef, opts)
}

func pushListFromStore(dockerCli command.Cli, targetRef reference.Named, opts pushOpts) error {
	manifests, err := dockerCli.ManifestStore().GetList(targetRef)
	if err != nil {
		return err
	}
	if len(manifests) == 0 {
		return errors.Errorf("%s not found", targetRef)
	}

	pushRequest, err := buildPushRequest(manifests, targetRef, opts.insecure)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := pushList(ctx, dockerCli, pushRequest); err != nil {
		return err
	}
	if opts.purge {
		return dockerCli.ManifestStore().Remove(targetRef)
	}
	return nil
}

func buildPushRequest(manifests []types.ImageManifest, targetRef reference.Named, insecure bool) (pushRequest, error) {
	req := pushRequest{targetRef: targetRef, insecure: insecure}

	var err error
	req.list, err = buildManifestList(manifests, targetRef)
	if err != nil {
		return req, err
	}

	targetRepo, err := registry.ParseRepositoryInfo(targetRef)
	if err != nil {
		return req, err
	}
	targetRepoName, err := registryclient.RepoNameForReference(targetRepo.Name)
	if err != nil {
		return req, err
	}

	for _, imageManifest := range manifests {
		manifestRepoName, err := registryclient.RepoNameForReference(imageManifest.Ref)
		if err != nil {
			return req, err
		}

		repoName, _ := reference.WithName(manifestRepoName)
		if repoName.Name() != targetRepoName {
			blobs, err := buildBlobRequestList(imageManifest, repoName)
			if err != nil {
				return req, err
			}
			req.manifestBlobs = append(req.manifestBlobs, blobs...)

			manifestPush, err := buildPutManifestRequest(imageManifest, targetRef)
			if err != nil {
				return req, err
			}
			req.mountRequests = append(req.mountRequests, manifestPush)
		}
	}
	return req, nil
}

func buildManifestList(manifests []types.ImageManifest, targetRef reference.Named) (*manifestlist.DeserializedManifestList, error) {
	targetRepoInfo, err := registry.ParseRepositoryInfo(targetRef)
	if err != nil {
		return nil, err
	}

	descriptors := []manifestlist.ManifestDescriptor{}
	for _, imageManifest := range manifests {
		if imageManifest.Descriptor.Platform == nil ||
			imageManifest.Descriptor.Platform.Architecture == "" ||
			imageManifest.Descriptor.Platform.OS == "" {
			return nil, errors.Errorf(
				"manifest %s must have an OS and Architecture to be pushed to a registry", imageManifest.Ref)
		}
		descriptor, err := buildManifestDescriptor(targetRepoInfo, imageManifest)
		if err != nil {
			return nil, err
		}
		descriptors = append(descriptors, descriptor)
	}

	return manifestlist.FromDescriptors(descriptors)
}

func buildManifestDescriptor(targetRepo *registry.RepositoryInfo, imageManifest types.ImageManifest) (manifestlist.ManifestDescriptor, error) {
	repoInfo, err := registry.ParseRepositoryInfo(imageManifest.Ref)
	if err != nil {
		return manifestlist.ManifestDescriptor{}, err
	}

	manifestRepoHostname := reference.Domain(repoInfo.Name)
	targetRepoHostname := reference.Domain(targetRepo.Name)
	if manifestRepoHostname != targetRepoHostname {
		return manifestlist.ManifestDescriptor{}, errors.Errorf("cannot use source images from a different registry than the target image: %s != %s", manifestRepoHostname, targetRepoHostname)
	}

	manifest := manifestlist.ManifestDescriptor{
		Descriptor: distribution.Descriptor{
			Digest:    imageManifest.Descriptor.Digest,
			Size:      imageManifest.Descriptor.Size,
			MediaType: imageManifest.Descriptor.MediaType,
		},
	}

	platform := types.PlatformSpecFromOCI(imageManifest.Descriptor.Platform)
	if platform != nil {
		manifest.Platform = *platform
	}

	if err = manifest.Descriptor.Digest.Validate(); err != nil {
		return manifestlist.ManifestDescriptor{}, errors.Wrapf(err,
			"digest parse of image %q failed", imageManifest.Ref)
	}

	return manifest, nil
}

func buildBlobRequestList(imageManifest types.ImageManifest, repoName reference.Named) ([]manifestBlob, error) {
	var blobReqs []manifestBlob

	for _, blobDigest := range imageManifest.Blobs() {
		canonical, err := reference.WithDigest(repoName, blobDigest)
		if err != nil {
			return nil, err
		}
		var os string
		if imageManifest.Descriptor.Platform != nil {
			os = imageManifest.Descriptor.Platform.OS
		}
		blobReqs = append(blobReqs, manifestBlob{canonical: canonical, os: os})
	}
	return blobReqs, nil
}

// nolint: interfacer
func buildPutManifestRequest(imageManifest types.ImageManifest, targetRef reference.Named) (mountRequest, error) {
	refWithoutTag, err := reference.WithName(targetRef.Name())
	if err != nil {
		return mountRequest{}, err
	}
	mountRef, err := reference.WithDigest(refWithoutTag, imageManifest.Descriptor.Digest)
	if err != nil {
		return mountRequest{}, err
	}

	// This indentation has to be added to ensure sha parity with the registry
	v2ManifestBytes, err := json.MarshalIndent(imageManifest.SchemaV2Manifest, "", "   ")
	if err != nil {
		return mountRequest{}, err
	}
	// indent only the DeserializedManifest portion of this, in order to maintain parity with the registry
	// and not alter the sha
	var v2Manifest schema2.DeserializedManifest
	if err = v2Manifest.UnmarshalJSON(v2ManifestBytes); err != nil {
		return mountRequest{}, err
	}
	imageManifest.SchemaV2Manifest = &v2Manifest

	return mountRequest{ref: mountRef, manifest: imageManifest}, err
}

func pushList(ctx context.Context, dockerCli command.Cli, req pushRequest) error {
	rclient := dockerCli.RegistryClient(req.insecure)

	if err := mountBlobs(ctx, rclient, req.targetRef, req.manifestBlobs); err != nil {
		return err
	}
	if err := pushReferences(ctx, dockerCli.Out(), rclient, req.mountRequests); err != nil {
		return err
	}
	dgst, err := rclient.PutManifest(ctx, req.targetRef, req.list)
	if err != nil {
		return err
	}

	fmt.Fprintln(dockerCli.Out(), dgst.String())
	return nil
}

func pushReferences(ctx context.Context, out io.Writer, client registryclient.RegistryClient, mounts []mountRequest) error {
	for _, mount := range mounts {
		newDigest, err := client.PutManifest(ctx, mount.ref, mount.manifest)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "Pushed ref %s with digest: %s\n", mount.ref, newDigest)
	}
	return nil
}

func mountBlobs(ctx context.Context, client registryclient.RegistryClient, ref reference.Named, blobs []manifestBlob) error {
	for _, blob := range blobs {
		err := client.MountBlob(ctx, blob.canonical, ref)
		switch err.(type) {
		case nil:
		case registryclient.ErrBlobCreated:
			if blob.os != "windows" {
				return fmt.Errorf("error mounting %s to %s", blob.canonical, ref)
			}
		default:
			return err
		}
	}
	return nil
}

func pushListFromYaml(dockerCli command.Cli, targetRef reference.Named, opts pushOpts) error {
	yamlInput, err := getYamlManifestList(opts.file)
	if err != nil {
		return err
	}
	if len(yamlInput.Manifests) == 0 {
		return errors.New("no manifests specified in file input")
	}
	ctx := context.Background()
	var manifests []types.ImageManifest
	for _, manifest := range yamlInput.Manifests {
		imageRef, err := normalizeReference(manifest.Image)
		if err != nil {
			return err
		}
		im, err := dockerCli.RegistryClient(opts.insecure).GetManifest(ctx, imageRef)
		if err != nil {
			return err
		}
		addYamlAnnotations(&im, manifest)
		if err := validateOSArch(im.Descriptor.Platform.OS, im.Descriptor.Platform.Architecture); err != nil {
			return err
		}
		manifests = append(manifests, im)
	}

	pushRequest, err := buildPushRequest(manifests, targetRef, opts.insecure)
	if err != nil {
		return err
	}
	return pushList(ctx, dockerCli, pushRequest)
}

func addYamlAnnotations(manifest *types.ImageManifest, ym yamlManifest) {
	if ym.Platform.Variant != "" {
		manifest.Descriptor.Platform.Variant = ym.Platform.Variant
	}
	if ym.Platform.OS != "" {
		manifest.Descriptor.Platform.OS = ym.Platform.OS
	}
	if ym.Platform.Architecture != "" {
		manifest.Descriptor.Platform.Architecture = ym.Platform.Architecture
	}
	if len(ym.Platform.OSFeatures) != 0 {
		manifest.Descriptor.Platform.OSFeatures = ym.Platform.OSFeatures
	}
}

func getYamlManifestList(yamlFile string) (yamlManifestList, error) {
	yamlBuf, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		return yamlManifestList{}, err
	}
	var yamlInput yamlManifestList
	return yamlInput, yaml.UnmarshalStrict(yamlBuf, &yamlInput)
}
