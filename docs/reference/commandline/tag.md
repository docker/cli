# tag

<!---MARKER_GEN_START-->
Create a tag TARGET_IMAGE that refers to SOURCE_IMAGE

### Aliases

`docker image tag`, `docker tag`


<!---MARKER_GEN_END-->

## Description

A full image name has the following format and components:

`[REGISTRY_HOSTNAME[:REGISTRY_PORT]/]REPOSITORY[:TAG]`

 - REGISTRY_HOSTNAME: The hostname must comply with standard DNS rules, but may
not contain underscores. If the hostname is not present, the command uses
Docker's public registry located at `registry-1.docker.io` by default.
 - REGISTRY_PORT: If a hostname is present, it may optionally be followed by a
port number in the format `:8080`.
- REPOSITORY: The repository, also referred to as the image name, is made up of
slash-separated components. Each name component may contain lowercase letters,
digits and separators. A separator is defined as a period, one or two
underscores, or one or more hyphens. A name component may not start or end with
a separator. When using Docker's public registry, the repository format is
`[USER_OR_ORGANIZATION/]REPOSITORY_NAME`. The USER_OR_ORGANIZATION is not used
for Docker Official Images.
 - TAG: A tag is a way to identify a specific version or variant of an image,
such as a particular release or build. A tag name must be valid ASCII and may
contain lowercase and uppercase letters, digits, underscores, periods and
hyphens. A tag name may not start with a period or a hyphen and may contain a
maximum of 128 characters.

You can group your images together using names and tags, and then upload them to
[share images on Docker
Hub](https://https://docs.docker.com/get-started/04_sharing_app/).

## Examples

### Tag an image referenced by ID

To tag a local image with ID "0e5574283393" into the "httpd" repository of the
"fedora" organization with "version1.0":

```console
$ docker tag 0e5574283393 fedora/httpd:version1.0
```

### Tag an image referenced by Name

To tag a local image with the name "httpd" into the "fedora" organization with
"version1.0":

```console
$ docker tag httpd fedora/httpd:version1.0
```

Note that since the tag name is not specified, the alias is created for an
existing local version `httpd:latest`.

### Tag an image referenced by Name and Tag

To tag a local image with the name "httpd" and the tag "test" into the "fedora"
organization with the tag "version1.0.test":

```console
$ docker tag httpd:test fedora/httpd:version1.0.test
```

### Tag an image for a private repository

To push an image to a private registry and not the public Docker registry you
must tag it with the registry hostname and port (if needed).

```console
$ docker tag 0e5574283393 myregistryhost:5000/fedora/httpd:version1.0
```