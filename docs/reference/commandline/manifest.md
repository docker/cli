---
title: "manifest"
description: "The manifest command description and usage"
keywords: "docker, manifest"
---

<!-- This file is maintained within the docker/cli Github
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

```markdown
Usage:  docker manifest COMMAND

Manage Docker image manifests and manifest lists

Options:
      --help   Print usage

Commands:
  annotate    Add additional information to a local image manifest
  create      Create a local manifest list for annotating and pushing to a registry
  inspect     Display an image manifest, or manifest list
  push        Push a manifest list to a repository

```

## Description

The `docker manifest` command by itself performs no action. In order to operate
on a manifest or manifest list, one of the subcommands must be used.

A single manifest is information about an image, such as layers, size, and digest.
The docker manifest command also gives users additional information such as the os
and architecture an image was built for.

A manifest list is a list of image layers that is created by specifying one or
more (ideally more than one) image names. It can then be used in the same way as
an image name in `docker pull` and `docker run` commands, for example.

Ideally a manifest list is created from images that are identical in function for
different os/arch combinations. For this reason, manifest lists are often referred to as
"multi-arch images". However, a user could create a manifest list that points
to two images -- one for windows on amd64, and one for darwin on amd64.

### manifest inspect

```
manifest inspect --help

Usage:  docker manifest inspect [OPTIONS] [MANIFEST_LIST] MANIFEST

Display an image manifest, or manifest list

Options:
      --help       Print usage
      --insecure   Allow communication with an insecure registry
  -v, --verbose    Output additional info including layers and platform
```

### manifest create 

```bash
Usage:  docker manifest create MANIFEST_LIST MANIFEST [MANIFEST...]

Create a local manifest list for annotating and pushing to a registry

Options:
  -a, --amend      Amend an existing manifest list
      --insecure   Allow communication with an insecure registry
      --help       Print usage
```

### manifest annotate
```bash
Usage:  docker manifest annotate [OPTIONS] MANIFEST_LIST MANIFEST

Add additional information to a local image manifest

Options:
      --arch string               Set architecture
      --help                      Print usage
      --os string                 Set operating system
      --os-features stringSlice   Set operating system feature
      --variant string            Set architecture variant

```

### manifest push
```bash
Usage:  docker manifest push [OPTIONS] MANIFEST_LIST

Push a manifest list to a repository

Options:
      --help       Print usage
      --insecure   Allow push to an insecure registry
  -p, --purge      Remove the local manifest list after push
```

### Working with insecure registries

The manifest command interacts solely with a Docker registry. Because of this, it has no way to query the engine for the list of allowed insecure registries. To allow the CLI to interact with an insecure registry, some `docker manifest` commands have an `--insecure` flag. For each transaction, such as a `create`, which queries a registry, the `--insecure` flag must be specified. This flag tells the CLI that this registry call may ignore security concerns like missing or self-signed certificates. Likewise, on a `manifest push` to an insecure registry, the `--insecure` flag must be specified. If this is not used with an insecure registry, the manifest command fails to find a registry that meets the default requirements.

## Examples

### Inspect an image's manifest object
 
```bash
$ docker manifest inspect hello-world
{
        "schemaVersion": 2,
        "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
        "config": {
                "mediaType": "application/vnd.docker.container.image.v1+json",
                "size": 1520,
                "digest": "sha256:1815c82652c03bfd8644afda26fb184f2ed891d921b20a0703b46768f9755c57"
        },
        "layers": [
                {
                        "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
                        "size": 972,
                        "digest": "sha256:b04784fba78d739b526e27edc02a5a8cd07b1052e9283f5fc155828f4b614c28"
                }
        ]
}
```

### Inspect an image's manifest and get the os/arch info

The `docker manifest inspect` command takes an optional `--verbose` flag
that gives you the image's name (Ref), and architecture and os (Platform).

Just as with other docker commands that take image names, you can refer to an image with or
without a tag, or by digest (e.g. hello-world@sha256:f3b3b28a45160805bb16542c9531888519430e9e6d6ffc09d72261b0d26ff74f).

Here is an example of inspecting an image's manifest with the `--verbose` flag:

```bash
$ docker manifest inspect --verbose hello-world
{
        "Ref": "docker.io/library/hello-world:latest",
        "Digest": "sha256:f3b3b28a45160805bb16542c9531888519430e9e6d6ffc09d72261b0d26ff74f",
        "SchemaV2Manifest": {
                "schemaVersion": 2,
                "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
                "config": {
                        "mediaType": "application/vnd.docker.container.image.v1+json",
                        "size": 1520,
                        "digest": "sha256:1815c82652c03bfd8644afda26fb184f2ed891d921b20a0703b46768f9755c57"
                },
                "layers": [
                        {
                                "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
                                "size": 972,
                                "digest": "sha256:b04784fba78d739b526e27edc02a5a8cd07b1052e9283f5fc155828f4b614c28"
                        }
                ]
        },
        "Platform": {
                "architecture": "amd64",
                "os": "linux"
        }
}
```

### Create and push a manifest list

To create a manifest list, you first `create` the manifest list locally by specifying the constituent images you would
like to have included in your manifest list. Keep in mind that this is pushed to a registry, so if you want to push
to a registry other than the docker registry, you need to create your manifest list with the registry name or IP and port.
This is similar to tagging an image and pushing it to a foreign registry.

After you have created your local copy of the manifest list, you may optionally
`annotate` it. Annotations allowed are the architecture and operating system (overriding the image's current values),
os features, and an archictecure variant. 

Finally, you need to `push` your manifest list to the desired registry. Below are descriptions of these three commands,
and an example putting them all together.

```bash
$ docker manifest create 45.55.81.106:5000/coolapp:v1 \
    45.55.81.106:5000/coolapp-ppc64le-linux:v1 \
    45.55.81.106:5000/coolapp-arm-linux:v1 \
    45.55.81.106:5000/coolapp-amd64-linux:v1 \
    45.55.81.106:5000/coolapp-amd64-windows:v1
Created manifest list 45.55.81.106:5000/coolapp:v1
```

```bash
$ docker manifest annotate 45.55.81.106:5000/coolapp:v1 45.55.81.106:5000/coolapp-arm-linux --arch arm
```

```bash
$ docker manifest push 45.55.81.106:5000/coolapp:v1
Pushed manifest 45.55.81.106:5000/coolapp@sha256:9701edc932223a66e49dd6c894a11db8c2cf4eccd1414f1ec105a623bf16b426 with digest: sha256:f67dcc5fc786f04f0743abfe0ee5dae9bd8caf8efa6c8144f7f2a43889dc513b
Pushed manifest 45.55.81.106:5000/coolapp@sha256:f3b3b28a45160805bb16542c9531888519430e9e6d6ffc09d72261b0d26ff74f with digest: sha256:b64ca0b60356a30971f098c92200b1271257f100a55b351e6bbe985638352f3a
Pushed manifest 45.55.81.106:5000/coolapp@sha256:39dc41c658cf25f33681a41310372f02728925a54aac3598310bfb1770615fc9 with digest: sha256:df436846483aff62bad830b730a0d3b77731bcf98ba5e470a8bbb8e9e346e4e8
Pushed manifest 45.55.81.106:5000/coolapp@sha256:f91b1145cd4ac800b28122313ae9e88ac340bb3f1e3a4cd3e59a3648650f3275 with digest: sha256:5bb8e50aa2edd408bdf3ddf61efb7338ff34a07b762992c9432f1c02fc0e5e62
sha256:050b213d49d7673ba35014f21454c573dcbec75254a08f4a7c34f66a47c06aba

```

### Inspect a manifest list

```bash
$ docker manifest inspect coolapp:v1
{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
   "manifests": [
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 424,
         "digest": "sha256:f67dcc5fc786f04f0743abfe0ee5dae9bd8caf8efa6c8144f7f2a43889dc513b",
         "platform": {
            "architecture": "arm",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 424,
         "digest": "sha256:b64ca0b60356a30971f098c92200b1271257f100a55b351e6bbe985638352f3a",
         "platform": {
            "architecture": "amd64",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 425,
         "digest": "sha256:df436846483aff62bad830b730a0d3b77731bcf98ba5e470a8bbb8e9e346e4e8",
         "platform": {
            "architecture": "ppc64le",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 425,
         "digest": "sha256:5bb8e50aa2edd408bdf3ddf61efb7338ff34a07b762992c9432f1c02fc0e5e62",
         "platform": {
            "architecture": "s390x",
            "os": "linux"
         }
      }
   ]
}
```

### Push to an insecure registry

Here is an example of creating and pushing a manifest list using a known insecure registry.

```
$ docker manifest create --insecure myprivateregistry.mycompany.com/repo/image:1.0 \
    myprivateregistry.mycompany.com/repo/image-linux-ppc64le:1.0 \
    myprivateregistry.mycompany.com/repo/image-linux-s390x:1.0 \
    myprivateregistry.mycompany.com/repo/image-linux-arm:1.0 \
    myprivateregistry.mycompany.com/repo/image-linux-armhf:1.0 \
    myprivateregistry.mycompany.com/repo/image-windows-amd64:1.0 \
    myprivateregistry.mycompany.com/repo/image-linux-amd64:1.0
```
```
$ docker manifest push --insecure myprivateregistry.mycompany.com/repo/image:tag
```

Note that the `--insecure` flag is not required to annotate a manifest list, since annotations are to a locally-stored copy of a manifest list. You may also skip the `--insecure` flag if you are performaing a `docker manifest inspect` on a locally-stored manifest list. Be sure to keep in mind that locally-stored manifest lists are never used by the engine on a `docker pull`.

