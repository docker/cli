# tag

<!---MARKER_GEN_START-->
Create a tag TARGET_IMAGE that refers to SOURCE_IMAGE

### Aliases

`docker image tag`, `docker tag`


<!---MARKER_GEN_END-->

## Description

A Docker image reference consists of several components that describe where the
image is stored and its identity. These components are:

```text
[HOST[:PORT]/]NAMESPACE/REPOSITORY[:TAG]
```

`HOST`
: Specifies the registry location where the image resides. If omitted, Docker
  defaults to Docker Hub (`docker.io`).

`PORT`
: An optional port number for the registry, if necessary (for example, `:5000`).

`NAMESPACE/REPOSITORY`
: The namespace (optional) usually represents a user or organization. The
  repository is required and identifies the specific image. If the namespace is
  omitted, Docker defaults to `library`, the namespace reserved for Docker
  Official Images.

`TAG`
: An optional identifier used to specify a particular version or variant of the
  image. If no tag is provided, Docker defaults to `latest`.

### Example image references

`example.com:5000/team/my-app:2.0`

- Host: `example.com`
- Port: `5000`
- Namespace: `team`
- Repository: `my-app`
- Tag: `2.0`

`alpine`

- Host: `docker.io` (default)
- Namespace: `library` (default)
- Repository: `alpine`
- Tag: `latest` (default)

For more information on the structure and rules of image naming, refer to the
[Distribution reference](https://pkg.go.dev/github.com/distribution/reference#pkg-overview)
as the canonical definition of the format.

## Examples

### Tag an image referenced by ID

To tag a local image with ID `0e5574283393` as `fedora/httpd` with the tag
`version1.0`:

```console
$ docker tag 0e5574283393 fedora/httpd:version1.0
```

### Tag an image referenced by Name

To tag a local image `httpd` as `fedora/httpd` with the tag `version1.0`:

```console
$ docker tag httpd fedora/httpd:version1.0
```

Note that since the tag name isn't specified, the alias is created for an
existing local version `httpd:latest`.

### Tag an image referenced by Name and Tag

To tag a local image with the name `httpd` and the tag `test` as `fedora/httpd`
with the tag `version1.0.test`:

```console
$ docker tag httpd:test fedora/httpd:version1.0.test
```

### Tag an image for a private registry

To push an image to a private registry and not the public Docker registry you
must include the registry hostname and port (if needed).

```console
$ docker tag 0e5574283393 myregistryhost:5000/fedora/httpd:version1.0
```
