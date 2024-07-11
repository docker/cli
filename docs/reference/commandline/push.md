# docker push

<!---MARKER_GEN_START-->
Upload an image to a registry

### Aliases

`docker image push`, `docker push`

### Options

| Name                      | Type     | Default | Description                                                                                                                                                                                                                                          |
|:--------------------------|:---------|:--------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-a`, `--all-tags`        | `bool`   |         | Push all tags of an image to the repository                                                                                                                                                                                                          |
| `--disable-content-trust` | `bool`   | `true`  | Skip image signing                                                                                                                                                                                                                                   |
| `--platform`              | `string` |         | Push a platform-specific manifest as a single-platform image to the registry.<br>Image index won't be pushed, meaning that other manifests, including attestations won't be preserved.<br>'os[/arch[/variant]]': Explicit platform (eg. linux/amd64) |
| `-q`, `--quiet`           | `bool`   |         | Suppress verbose output                                                                                                                                                                                                                              |


<!---MARKER_GEN_END-->

