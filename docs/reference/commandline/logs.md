# logs

<!---MARKER_GEN_START-->
Fetch the logs of a container

### Aliases

`docker container logs`, `docker logs`

### Options

| Name                 | Type     | Default | Description                                                                                                                    |
|:---------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------|
| `--details`          |          |         | Show extra details provided to logs                                                                                            |
| `-f`, `--follow`     |          |         | Follow log output                                                                                                              |
| [`--since`](#since)  | `string` |         | Show logs after a [timestamp](#date_formats) (e.g. `2013-01-02T13:23:37Z`) or relative to now (e.g. `42m` for 42 minutes ago)  |
| `-n`, `--tail`       | `string` | `all`   | Number of lines to show from the end of the logs                                                                               |
| `-t`, `--timestamps` |          |         | Show timestamps                                                                                                                |
| [`--until`](#until)  | `string` |         | Show logs before a [timestamp](#date_formats) (e.g. `2013-01-02T13:23:37Z`) or relative to now (e.g. `42m` for 42 minutes ago) |


<!---MARKER_GEN_END-->

## Description

The `docker logs` command batch-retrieves logs present at the time of execution.

For more information about selecting and configuring logging drivers, refer to
[Configure logging drivers](https://docs.docker.com/config/containers/logging/configure/).

The `docker logs --follow` command will continue streaming the new output from
the container's `STDOUT` and `STDERR`.

Passing a negative number or a non-integer to `--tail` is invalid and the
value is set to `all` in that case.

The `docker logs --timestamps` command will add an [RFC3339Nano timestamp](https://pkg.go.dev/time#RFC3339Nano)
, for example `2014-09-16T06:17:46.000000000Z`, to each
log entry. To ensure that the timestamps are aligned the
nano-second part of the timestamp will be padded with zero when necessary.

The `docker logs --details` command will add on extra attributes, such as
environment variables and labels, provided to `--log-opt` when creating the
container.

The `--since` option shows only the container logs generated **after**
a given [timestamp](#date_formats). You can combine the `--since` option with
either or both of the `--follow` or `--tail` options.

The `--until` option shows only the container logs generated **before** a given
[timestamp](#date_formats). You can combine the `--since` option with either or
both of the `--follow` (for timestamps in the future) or `--tail` options.

### <a name="date_formats"></a> Supported timestamp formats

The date values for `--since/--until` can be specified in the following ways:
* an RFC3339 or RFC3339Nano date string.
* date strings of the following formats: `2006-01-02`, `2006-01-02Z07:00`,
  `2006-01-02T15:04:05`, and `2006-01-02T15:04:05.999999999`. Note that if you
  do not provide either a `Z` or a `+-00:00` timezone offset at the end of the
  timestamp, the local timezone on the client will be used.
* Unix timestamps of the form `seconds[.nanoseconds]`, representing the number
  of seconds that have elapsed since January 1, 1970 (midnight UTC/GMT),
  not counting leap seconds (aka Unix epoch or Unix time) and an optional
  `.nanoseconds` field with no more than nine digits for representing
  fractions of a second.
* a Go duration string (e.g. `90s`, `1.5m`, `1h30m`) which is subtracted from
  the Docker client's current time to get the equivalent timestamp. Specifying
  a negative value (e.g. `-60m`) will lead to a timestamp in the future.
  Accepts all time measurement units accepted by [`time.ParseDuration()`](https://pkg.go.dev/time#ParseDuration).


## Examples

### <a name="until"></a> Retrieve logs until a specific point in time (--until)

In order to retrieve logs before a specific point in time, run:

```console
$ docker run --name test -d busybox sh -c "while true; do $(echo date); sleep 1; done"
$ sleep 5
$ date
Tue 14 Nov 2017 16:40:00 CET
$ docker logs --until=3s test
Tue 14 Nov 2017 16:39:55 CET
Tue 14 Nov 2017 16:39:56 CET
Tue 14 Nov 2017 16:39:57 CET
<EOF>
```

### <a name="since"></a> Follow logs after a specific point in time (--since)

In order to follow logs after a specific point in time, run:

```console
$ docker run --name test -d busybox sh -c "while true; do $(echo date); sleep 1; done"
$ sleep 5
$ date
Tue 14 Nov 2017 16:40:00 CET
$ docker logs -f --since=3s test
Tue 14 Nov 2017 16:39:57 CET
Tue 14 Nov 2017 16:39:58 CET
Tue 14 Nov 2017 16:39:59 CET
Tue 14 Nov 2017 16:40:00 CET
Tue 14 Nov 2017 16:40:01 CET
...
```
