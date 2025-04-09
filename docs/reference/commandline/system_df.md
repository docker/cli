# system df

<!---MARKER_GEN_START-->
Show docker disk usage

### Options

| Name                  | Type     | Default | Description                                                                                                                                                                                                                                                                                                                                                                                                                          |
|:----------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`--format`](#format) | `string` |         | Format output using a custom template:<br>'table':            Print output in table format with column headers (default)<br>'table TEMPLATE':   Print output in table format using the given Go template<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |
| `-v`, `--verbose`     | `bool`   |         | Show detailed information on space usage                                                                                                                                                                                                                                                                                                                                                                                             |


<!---MARKER_GEN_END-->

## Description

The `docker system df` command displays information regarding the
amount of disk space used by the Docker daemon.

## Examples

By default the command displays a summary of the data used:

```console
$ docker system df

TYPE                TOTAL               ACTIVE              SIZE                RECLAIMABLE
Images              5                   2                   16.43 MB            11.63 MB (70%)
Containers          2                   0                   212 B               212 B (100%)
Local Volumes       2                   1                   36 B                0 B (0%)
```

Use the `-v, --verbose` flag to get more detailed information:

```console
$ docker system df -v

Images space usage:

REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE                SHARED SIZE         UNIQUE SIZE         CONTAINERS
my-curl             latest              b2789dd875bf        6 minutes ago       11 MB               11 MB               5 B                 0
my-jq               latest              ae67841be6d0        6 minutes ago       9.623 MB            8.991 MB            632.1 kB            0
<none>              <none>              a0971c4015c1        6 minutes ago       11 MB               11 MB               0 B                 0
alpine              latest              4e38e38c8ce0        9 weeks ago         4.799 MB            0 B                 4.799 MB            1
alpine              3.3                 47cf20d8c26c        9 weeks ago         4.797 MB            4.797 MB            0 B                 1

Containers space usage:

CONTAINER ID        IMAGE               COMMAND             LOCAL VOLUMES       SIZE                CREATED             STATUS                      NAMES
4a7f7eebae0f        alpine:latest       "sh"                1                   0 B                 16 minutes ago      Exited (0) 5 minutes ago    hopeful_yalow
f98f9c2aa1ea        alpine:3.3          "sh"                1                   212 B               16 minutes ago      Exited (0) 48 seconds ago   anon-vol

Local Volumes space usage:

NAME                                                               LINKS               SIZE
07c7bdf3e34ab76d921894c2b834f073721fccfbbcba792aa7648e3a7a664c2e   2                   36 B
my-named-vol                                                       0                   0 B
```

* `SHARED SIZE` is the amount of space that an image shares with another one (i.e. their common data)
* `UNIQUE SIZE` is the amount of space that's only used by a given image
* `SIZE` is the virtual size of the image, it's the sum of `SHARED SIZE` and `UNIQUE SIZE`

> [!NOTE]
> Network information isn't shown, because it doesn't consume disk space.

## Performance

Running the `system df` command can be resource-intensive. It traverses the
filesystem of every image, container, and volume in the system. You should be
careful running this command in systems with lots of images, containers, or
volumes or in systems where some images, containers, or volumes have large
filesystems with many files. You should also be careful not to run this command
in systems where performance is critical.

### <a name="format"></a> Format the output (--format)

The formatting option (`--format`) pretty prints the disk usage output
using a Go template.

Valid placeholders for the Go template are listed below:

| Placeholder    | Description                                |
|----------------|--------------------------------------------|
| `.Type`        | `Images`, `Containers` and `Local Volumes` |
| `.TotalCount`  | Total number of items                      |
| `.Active`      | Number of active items                     |
| `.Size`        | Available size                             |
| `.Reclaimable` | Reclaimable size                           |

When using the `--format` option, the `system df` command outputs
the data exactly as the template declares or, when using the
`table` directive, includes column headers as well.

The following example uses a template without headers and outputs the
`Type` and `TotalCount` entries separated by a colon (`:`):

```console
$ docker system df --format "{{.Type}}: {{.TotalCount}}"

Images: 2
Containers: 4
Local Volumes: 1
```

To list the disk usage with size and reclaimable size in a table format you
can use:

```console
$ docker system df --format "table {{.Type}}\t{{.Size}}\t{{.Reclaimable}}"

TYPE                SIZE                RECLAIMABLE
Images              2.547 GB            2.342 GB (91%)
Containers          0 B                 0 B
Local Volumes       150.3 MB            150.3 MB (100%)
<Paste>
```

To list all information in JSON format, use the `json` directive:

```console
$ docker system df --format json
{"Active":"2","Reclaimable":"2.498GB (94%)","Size":"2.631GB","TotalCount":"6","Type":"Images"}
{"Active":"1","Reclaimable":"1.114kB (49%)","Size":"2.23kB","TotalCount":"7","Type":"Containers"}
{"Active":"0","Reclaimable":"256.5MB (100%)","Size":"256.5MB","TotalCount":"1","Type":"Local Volumes"}
{"Active":"0","Reclaimable":"158B","Size":"158B","TotalCount":"17","Type":"Build Cache"}
```

The format option has no effect when the `--verbose` option is used.

## Related commands
* [system prune](system_prune.md)
* [container prune](container_prune.md)
* [volume prune](volume_prune.md)
* [image prune](image_prune.md)
* [network prune](network_prune.md)
