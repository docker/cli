# docker load

<!---MARKER_GEN_START-->
Load an image from a tar archive or STDIN

### Aliases

`docker image load`, `docker load`

### Options

| Name            | Type     | Default | Description                                                                                                                                                                                  |
|:----------------|:---------|:--------|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-i`, `--input` | `string` |         | Read from tar archive file, instead of STDIN                                                                                                                                                 |
| `--platform`    | `string` |         | Pick a single-platform to be loaded if the image is multi-platform.<br>Full multi-platform image will be load if not specified.<br><br>Format: os[/arch[/variant]]<br>Example: `linux/amd64` |
| `-q`, `--quiet` | `bool`   |         | Suppress the load output                                                                                                                                                                     |


<!---MARKER_GEN_END-->

