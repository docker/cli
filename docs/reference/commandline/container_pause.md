# pause

<!---MARKER_GEN_START-->
Pause all processes within one or more containers

### Aliases

`docker container pause`, `docker pause`


<!---MARKER_GEN_END-->

## Description

The `docker pause` command suspends all processes in the specified containers.
On Linux, this uses the freezer cgroup. Traditionally, when suspending a process
the `SIGSTOP` signal is used, which is observable by the process being suspended.
With the freezer cgroup the process is unaware, and unable to capture,
that it is being suspended, and subsequently resumed. On Windows, only Hyper-V
containers can be paused.

See the
[freezer cgroup documentation](https://www.kernel.org/doc/Documentation/cgroup-v1/freezer-subsystem.txt)
for further details.

## Examples

```console
$ docker pause my_container
```

## Related commands

* [unpause](unpause.md)
