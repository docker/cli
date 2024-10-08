The **docker container logs** command batch-retrieves whatever logs are present for
a container at the time of execution. This does not guarantee execution
order when combined with a docker run (i.e., your run may not have generated
any logs at the time you execute docker container logs).

The **docker container logs --follow** command combines commands **docker container logs** and
**docker attach**. It will first return all logs from the beginning and
then continue streaming new output from the container's stdout and stderr.

**Warning**: This command works only for the **json-file** or **journald**
logging drivers.

The `--since` and `--until` options can be Unix timestamps, date formatted timestamps, 
or Go duration strings supported by [ParseDuration](https://pkg.go.dev/time#ParseDuration) (e.g. `10m`, `1h30m`) computed relative to the client machine's
time. Supported formats for date formatted time stamps include RFC3339Nano,
RFC3339, `2006-01-02T15:04:05`, `2006-01-02T15:04:05.999999999`,
`2006-01-02T07:00`, and `2006-01-02`. The local timezone on the client will be
used if you do not provide either a `Z` or a `+-00:00` timezone offset at the
end of the timestamp.  When providing Unix timestamps enter
seconds[.nanoseconds], where seconds is the number of seconds that have elapsed
since January 1, 1970 (midnight UTC/GMT), not counting leap  seconds (aka Unix
epoch or Unix time), and the optional .nanoseconds field is a fraction of a
second no more than nine digits long. You can combine the `--since` or `--until` 
options with either or both of the `--follow` or `--tail` options.

The `docker container logs --details` command will add on extra attributes, such as
environment variables and labels, provided to `--log-opt` when creating the
container.

In order to retrieve logs before a specific point in time, run:

```console
$ docker run --name test -d busybox sh -c "while true; do $(echo date); sleep 1; done"
$ date
Tue 14 Nov 2017 16:40:00 CET
$ docker logs -f --until=2s test
Tue 14 Nov 2017 16:40:00 CET
Tue 14 Nov 2017 16:40:01 CET
Tue 14 Nov 2017 16:40:02 CET
```
