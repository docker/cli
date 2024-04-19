# stack deploy

<!---MARKER_GEN_START-->
Deploy a new stack or update an existing stack

### Aliases

`docker stack deploy`, `docker stack up`

### Options

| Name                                                     | Type          | Default  | Description                                                                                       |
|:---------------------------------------------------------|:--------------|:---------|:--------------------------------------------------------------------------------------------------|
| [`-c`](#compose-file), [`--compose-file`](#compose-file) | `stringSlice` |          | Path to a Compose file, or `-` to read from stdin                                                 |
| `-d`, `--detach`                                         | `bool`        | `true`   | Exit immediately instead of waiting for the stack services to converge                            |
| `--prune`                                                |               |          | Prune services that are no longer referenced                                                      |
| `-q`, `--quiet`                                          |               |          | Suppress progress output                                                                          |
| `--resolve-image`                                        | `string`      | `always` | Query the registry to resolve image digest and supported platforms (`always`, `changed`, `never`) |
| `--with-registry-auth`                                   |               |          | Send registry authentication details to Swarm agents                                              |


<!---MARKER_GEN_END-->

## Description

Create and update a stack from a `compose` file on the swarm.

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### <a name="compose-file"></a> Compose file (--compose-file)

The `deploy` command supports Compose file version `3.0` and above.

```console
$ docker stack deploy --compose-file docker-compose.yml vossibility

Ignoring unsupported options: links

Creating network vossibility_vossibility
Creating network vossibility_default
Creating service vossibility_nsqd
Creating service vossibility_logstash
Creating service vossibility_elasticsearch
Creating service vossibility_kibana
Creating service vossibility_ghollector
Creating service vossibility_lookupd
```

The Compose file can also be provided as standard input with `--compose-file -`:

```console
$ cat docker-compose.yml | docker stack deploy --compose-file - vossibility

Ignoring unsupported options: links

Creating network vossibility_vossibility
Creating network vossibility_default
Creating service vossibility_nsqd
Creating service vossibility_logstash
Creating service vossibility_elasticsearch
Creating service vossibility_kibana
Creating service vossibility_ghollector
Creating service vossibility_lookupd
```

If your configuration is split between multiple Compose files, e.g. a base
configuration and environment-specific overrides, you can provide multiple
`--compose-file` flags.

```console
$ docker stack deploy --compose-file docker-compose.yml -c docker-compose.prod.yml vossibility

Ignoring unsupported options: links

Creating network vossibility_vossibility
Creating network vossibility_default
Creating service vossibility_nsqd
Creating service vossibility_logstash
Creating service vossibility_elasticsearch
Creating service vossibility_kibana
Creating service vossibility_ghollector
Creating service vossibility_lookupd
```

You can verify that the services were correctly created:

```console
$ docker service ls

ID            NAME                               MODE        REPLICAS  IMAGE
29bv0vnlm903  vossibility_lookupd                replicated  1/1       nsqio/nsq@sha256:eeba05599f31eba418e96e71e0984c3dc96963ceb66924dd37a47bf7ce18a662
4awt47624qwh  vossibility_nsqd                   replicated  1/1       nsqio/nsq@sha256:eeba05599f31eba418e96e71e0984c3dc96963ceb66924dd37a47bf7ce18a662
4tjx9biia6fs  vossibility_elasticsearch          replicated  1/1       elasticsearch@sha256:12ac7c6af55d001f71800b83ba91a04f716e58d82e748fa6e5a7359eed2301aa
7563uuzr9eys  vossibility_kibana                 replicated  1/1       kibana@sha256:6995a2d25709a62694a937b8a529ff36da92ebee74bafd7bf00e6caf6db2eb03
9gc5m4met4he  vossibility_logstash               replicated  1/1       logstash@sha256:2dc8bddd1bb4a5a34e8ebaf73749f6413c101b2edef6617f2f7713926d2141fe
axqh55ipl40h  vossibility_vossibility-collector  replicated  1/1       icecrime/vossibility-collector@sha256:f03f2977203ba6253988c18d04061c5ec7aab46bca9dfd89a9a1fa4500989fba
```

## Related commands

* [stack ls](stack_ls.md)
* [stack ps](stack_ps.md)
* [stack rm](stack_rm.md)
* [stack services](stack_services.md)
* [stack config](stack_config.md)
