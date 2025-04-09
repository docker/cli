# swarm update

<!---MARKER_GEN_START-->
Update the swarm

### Options

| Name                     | Type          | Default     | Description                                                 |
|:-------------------------|:--------------|:------------|:------------------------------------------------------------|
| `--autolock`             | `bool`        |             | Change manager autolocking setting (true\|false)            |
| `--cert-expiry`          | `duration`    | `2160h0m0s` | Validity period for node certificates (ns\|us\|ms\|s\|m\|h) |
| `--dispatcher-heartbeat` | `duration`    | `5s`        | Dispatcher heartbeat period (ns\|us\|ms\|s\|m\|h)           |
| `--external-ca`          | `external-ca` |             | Specifications of one or more certificate signing endpoints |
| `--max-snapshots`        | `uint64`      | `0`         | Number of additional Raft snapshots to retain               |
| `--snapshot-interval`    | `uint64`      | `10000`     | Number of log entries between Raft snapshots                |
| `--task-history-limit`   | `int64`       | `5`         | Task history retention limit                                |


<!---MARKER_GEN_END-->

## Description

Updates a swarm with new parameter values.

> [!NOTE]
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

```console
$ docker swarm update --cert-expiry 720h
```

## Related commands

* [swarm ca](swarm_ca.md)
* [swarm init](swarm_init.md)
* [swarm join](swarm_join.md)
* [swarm join-token](swarm_join-token.md)
* [swarm leave](swarm_leave.md)
* [swarm unlock](swarm_unlock.md)
* [swarm unlock-key](swarm_unlock-key.md)
