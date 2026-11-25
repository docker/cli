# manifest annotate

<!---MARKER_GEN_START-->
Add additional information to a local image manifest

### Options

| Name            | Type          | Default | Description                  |
|:----------------|:--------------|:--------|:-----------------------------|
| `--arch`        | `string`      |         | Set architecture             |
| `--os`          | `string`      |         | Set operating system         |
| `--os-features` | `stringSlice` |         | Set operating system feature |
| `--os-version`  | `string`      |         | Set operating system version |
| `--variant`     | `string`      |         | Set architecture variant     |


<!---MARKER_GEN_END-->

## Description

`--os`, `--arch`, and `--variant` take OCI platform values, not distro nicknames
like `armhf` or `armv7`.

Common `--os` values are `linux` and `windows`. Common `--arch` values are
`amd64`, `arm64`, `arm`, `386`, `ppc64le`, `s390x`, and `riscv64`.

`--variant` is mostly for 32-bit ARM. An `armhf` / `armv7` image is
`--arch arm --variant v7`. `arm64` usually has no variant (or `v8`).


