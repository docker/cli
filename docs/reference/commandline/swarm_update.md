---
title: "swarm update"
description: "The swarm update command description and usage"
keywords: "swarm, update"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# swarm update

```markdown
Usage:  docker swarm update [OPTIONS]

Update the swarm

Options:
      --autolock                        Change manager autolocking setting (true|false)
      --cert-expiry duration            Validity period for node certificates (ns|us|ms|s|m|h) (default 2160h0m0s)
      --dispatcher-heartbeat duration   Dispatcher heartbeat period (ns|us|ms|s|m|h) (default 5s)
      --external-ca external-ca         Specifications of one or more certificate signing endpoints in the form: "protocol=<protocol>,url=<url>"
      --help                            Print usage
      --max-snapshots uint              Number of additional Raft snapshots to retain
      --snapshot-interval uint          Number of log entries between Raft snapshots (default 10000)
      --task-history-limit int          Task history retention limit (default 5)
```

## Description

Updates a swarm with new parameter values. This command must target a manager node.

## Examples

```bash
$ docker swarm update --cert-expiry 720h
```

## `--external-ca`

Provide an external CA URL to the managers - this CA will be used to issue all new
node certificates.  One or more of these flags can be passed in order to provide
multiple alternative URLs - the swarm will try each one in order before falling
back on the next. If an external CA URL is added, swarm will only use the external CA
even if it has access to the CA key.

The only protocol currently supported for external CAs is the CFSSL signing
protocol, and the URL provided should be to the [`/api/v1/cfssl/sign`](https://github.com/cloudflare/cfssl/blob/master/doc/api/endpoint_sign.txt)
endpoint on a CFSSL server.

The URL must be HTTPS, and the TLS certificate for the external URL must be signed
by the CA certificate currently used by the swarm.

Example usage:

`--external-ca=protocol=cfssl,url=https://my.cfssl.server/api/v1/cfssl/sign`

External CAs can be removed though by passing an empty string, e.g.
`--external-ca=""`.

## Related commands

* [swarm ca](swarm_ca.md)
* [swarm init](swarm_init.md)
* [swarm join](swarm_join.md)
* [swarm join-token](swarm_join_token.md)
* [swarm leave](swarm_leave.md)
* [swarm unlock](swarm_unlock.md)
* [swarm unlock-key](swarm_unlock_key.md)
