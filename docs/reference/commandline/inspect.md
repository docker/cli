# inspect

<!---MARKER_GEN_START-->
Return low-level information on Docker objects

### Options

| Name                                   | Type     | Default | Description                                                                                                                                                                                                                                                        |
|:---------------------------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`-f`](#format), [`--format`](#format) | `string` |         | Format output using a custom template:<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |
| [`-s`](#size), [`--size`](#size)       | `bool`   |         | Display total file sizes if the type is container                                                                                                                                                                                                                  |
| [`--type`](#type)                      | `string` |         | Return JSON for specified type                                                                                                                                                                                                                                     |


<!---MARKER_GEN_END-->

## Description

Docker inspect provides detailed information on constructs controlled by Docker.

By default, `docker inspect` will render results in a JSON array.

### <a name="format"></a> Format the output (--format)

If a format is specified, the given template will be executed for each result.

Go's [text/template](https://pkg.go.dev/text/template) package describes
all the details of the format.

### <a name="type"></a> Specify target type (--type)

`--type config|container|image|node|network|secret|service|volume|task|plugin`

The `docker inspect` command matches any type of object by either ID or name. In
some cases multiple type of objects (for example, a container and a volume)
exist with the same name, making the result ambiguous.

To restrict `docker inspect` to a specific type of object, use the `--type`
option.

The following example inspects a volume named `myvolume`.

```console
$ docker inspect --type=volume myvolume
```

### <a name="size"></a> Inspect the size of a container (-s, --size)

The `--size`, or short-form `-s`, option adds two additional fields to the
`docker inspect` output. This option only works for containers. The container
doesn't have to be running, it also works for stopped containers.

```console
$ docker inspect --size mycontainer
```

The output includes the full output of a regular `docker inspect` command, with
the following additional fields:

- `SizeRootFs`: the total size of all the files in the container, in bytes.
- `SizeRw`: the size of the files that have been created or changed in the
  container, compared to it's image, in bytes.

```console
$ docker run --name database -d redis
3b2cbf074c99db4a0cad35966a9e24d7bc277f5565c17233386589029b7db273
$ docker inspect --size database -f '{{ .SizeRootFs }}'
123125760
$ docker inspect --size database -f '{{ .SizeRw }}'
8192
$ docker exec database fallocate -l 1000 /newfile
$ docker inspect --size database -f '{{ .SizeRw }}'
12288
```

## Examples

### Get an instance's IP address

For the most part, you can pick out any field from the JSON in a fairly
straightforward manner.

```console
$ docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $INSTANCE_ID
```

### Get an instance's MAC address

```console
$ docker inspect --format='{{range .NetworkSettings.Networks}}{{.MacAddress}}{{end}}' $INSTANCE_ID
```

### Get an instance's log path

```console
$ docker inspect --format='{{.LogPath}}' $INSTANCE_ID
```

### Get an instance's image name

```console
$ docker inspect --format='{{.Config.Image}}' $INSTANCE_ID
```

### List all port bindings

You can loop over arrays and maps in the results to produce simple text output:

```console
$ docker inspect --format='{{range $p, $conf := .NetworkSettings.Ports}} {{$p}} -> {{(index $conf 0).HostPort}} {{end}}' $INSTANCE_ID
```

### Find a specific port mapping

The `.Field` syntax doesn't work when the field name begins with a number, but
the template language's `index` function does. The `.NetworkSettings.Ports`
section contains a map of the internal port mappings to a list of external
address/port objects. To grab just the numeric public port, you use `index` to
find the specific port map, and then `index` 0 contains the first object inside
of that. Then, specify the `HostPort` field to get the public address.

```console
$ docker inspect --format='{{(index (index .NetworkSettings.Ports "8787/tcp") 0).HostPort}}' $INSTANCE_ID
```

### Get a subsection in JSON format

If you request a field which is itself a structure containing other fields, by
default you get a Go-style dump of the inner values. Docker adds a template
function, `json`, which can be applied to get results in JSON format.

```console
$ docker inspect --format='{{json .Config}}' $INSTANCE_ID
```
