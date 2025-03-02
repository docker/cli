# swarm unlock-key

<!---MARKER_GEN_START-->
Manage the unlock key

### Options

| Name                                | Type   | Default | Description        |
|:------------------------------------|:-------|:--------|:-------------------|
| [`-q`](#quiet), [`--quiet`](#quiet) | `bool` |         | Only display token |
| [`--rotate`](#rotate)               | `bool` |         | Rotate unlock key  |


<!---MARKER_GEN_END-->

## Description

An unlock key is a secret key needed to unlock a manager after its Docker daemon
restarts. These keys are only used when the autolock feature is enabled for the
swarm.

You can view or rotate the unlock key using `swarm unlock-key`. To view the key,
run the `docker swarm unlock-key` command without any arguments:

> [!NOTE]
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

```console
$ docker swarm unlock-key

To unlock a swarm manager after it restarts, run the `docker swarm unlock`
command and provide the following key:

    SWMKEY-1-fySn8TY4w5lKcWcJPIpKufejh9hxx5KYwx6XZigx3Q4

Remember to store this key in a password manager, since without it you
will not be able to restart the manager.
```

Use the `--rotate` flag to rotate the unlock key to a new, randomly-generated
key:

```console
$ docker swarm unlock-key --rotate

Successfully rotated manager unlock key.

To unlock a swarm manager after it restarts, run the `docker swarm unlock`
command and provide the following key:

    SWMKEY-1-7c37Cc8654o6p38HnroywCi19pllOnGtbdZEgtKxZu8

Remember to store this key in a password manager, since without it you
will not be able to restart the manager.
```

The `-q` (or `--quiet`) flag only prints the key:

```console
$ docker swarm unlock-key -q

SWMKEY-1-7c37Cc8654o6p38HnroywCi19pllOnGtbdZEgtKxZu8
```

### <a name="rotate"></a> `--rotate`

This flag rotates the unlock key, replacing it with a new randomly-generated
key. The old unlock key will no longer be accepted.

### <a name="quiet"></a> `--quiet`

Only print the unlock key, without instructions.

## Related commands

* [swarm ca](swarm_ca.md)
* [swarm init](swarm_init.md)
* [swarm join](swarm_join.md)
* [swarm join-token](swarm_join-token.md)
* [swarm leave](swarm_leave.md)
* [swarm unlock](swarm_unlock.md)
* [swarm update](swarm_update.md)
