# swarm init

<!---MARKER_GEN_START-->
Initialize a swarm

### Options

| Name                                        | Type          | Default        | Description                                                                                                                  |
|:--------------------------------------------|:--------------|:---------------|:-----------------------------------------------------------------------------------------------------------------------------|
| [`--advertise-addr`](#advertise-addr)       | `string`      |                | Advertised address (format: `<ip\|interface>[:port]`)                                                                        |
| [`--autolock`](#autolock)                   |               |                | Enable manager autolocking (requiring an unlock key to start a stopped manager)                                              |
| [`--availability`](#availability)           | `string`      | `active`       | Availability of the node (`active`, `pause`, `drain`)                                                                        |
| `--cert-expiry`                             | `duration`    | `2160h0m0s`    | Validity period for node certificates (ns\|us\|ms\|s\|m\|h)                                                                  |
| [`--data-path-addr`](#data-path-addr)       | `string`      |                | Address or interface to use for data path traffic (format: `<ip\|interface>`)                                                |
| [`--data-path-port`](#data-path-port)       | `uint32`      | `0`            | Port number to use for data path traffic (1024 - 49151). If no value is set or is set to 0, the default port (4789) is used. |
| [`--default-addr-pool`](#default-addr-pool) | `ipNetSlice`  |                | default address pool in CIDR format                                                                                          |
| `--default-addr-pool-mask-length`           | `uint32`      | `24`           | default address pool subnet mask length                                                                                      |
| `--dispatcher-heartbeat`                    | `duration`    | `5s`           | Dispatcher heartbeat period (ns\|us\|ms\|s\|m\|h)                                                                            |
| [`--external-ca`](#external-ca)             | `external-ca` |                | Specifications of one or more certificate signing endpoints                                                                  |
| [`--force-new-cluster`](#force-new-cluster) |               |                | Force create a new cluster from current state                                                                                |
| [`--listen-addr`](#listen-addr)             | `node-addr`   | `0.0.0.0:2377` | Listen address (format: `<ip\|interface>[:port]`)                                                                            |
| [`--max-snapshots`](#max-snapshots)         | `uint64`      | `0`            | Number of additional Raft snapshots to retain                                                                                |
| [`--snapshot-interval`](#snapshot-interval) | `uint64`      | `10000`        | Number of log entries between Raft snapshots                                                                                 |
| `--task-history-limit`                      | `int64`       | `5`            | Task history retention limit                                                                                                 |


<!---MARKER_GEN_END-->

## Description

Initialize a swarm. The Docker Engine targeted by this command becomes a manager
in the newly created single-node swarm.

## Examples

```console
$ docker swarm init --advertise-addr 192.168.99.121

Swarm initialized: current node (bvz81updecsj6wjz393c09vti) is now a manager.

To add a worker to this swarm, run the following command:

    docker swarm join --token SWMTKN-1-3pu6hszjas19xyp7ghgosyx9k8atbfcr8p2is99znpy26u2lkl-1awxwuwd3z9j1z3puu7rcgdbx 172.17.0.2:2377

To add a manager to this swarm, run 'docker swarm join-token manager' and follow the instructions.
```

The `docker swarm init` command generates two random tokens: a worker token and
a manager token. When you join a new node to the swarm, the node joins as a
worker or manager node based upon the token you pass to [swarm
join](swarm_join.md).

After you create the swarm, you can display or rotate the token using
[swarm join-token](swarm_join-token.md).

### <a name="autolock"></a> Protect manager keys and data (--autolock)

The `--autolock` flag enables automatic locking of managers with an encryption
key. The private keys and data stored by all managers are protected by the
encryption key printed in the output, and is inaccessible without it. Make sure
to store this key securely, in order to reactivate a manager after it restarts.
Pass the key to the `docker swarm unlock` command to reactivate the manager.
You can disable autolock by running `docker swarm update --autolock=false`.
After disabling it, the encryption key is no longer required to start the
manager, and it will start up on its own without user intervention.

### <a name=""></a> Configure node healthcheck frequency (--dispatcher-heartbeat)

The `--dispatcher-heartbeat` flag sets the frequency at which nodes are told to
report their health.

### <a name="external-ca"></a> Use an external certificate authority (--external-ca)

This flag sets up the swarm to use an external CA to issue node certificates.
The value takes the form `protocol=X,url=Y`. The value for `protocol` specifies
what protocol should be used to send signing requests to the external CA.
Currently, the only supported value is `cfssl`. The URL specifies the endpoint
where signing requests should be submitted.

### <a name="force-new-cluster"></a> Force-restart node as a single-mode manager (--force-new-cluster)

This flag forces an existing node that was part of a quorum that was lost to
restart as a single-node Manager without losing its data.

### <a name="listen-addr"></a> Specify interface for inbound control plane traffic (--listen-addr)

The node listens for inbound swarm manager traffic on this address. The default
is to listen on `0.0.0.0:2377`. It is also possible to specify a network
interface to listen on that interface's address; for example `--listen-addr
eth0:2377`.

Specifying a port is optional. If the value is a bare IP address or interface
name, the default port 2377 is used.

### <a name="advertise-addr"></a> Specify interface for outbound control plane traffic (--advertise-addr)

The `--advertise-addr` flag specifies the address that will be advertised to
other members of the swarm for API access and overlay networking. If
unspecified, Docker will check if the system has a single IP address, and use
that IP address with the listening port (see `--listen-addr`). If the system
has multiple IP addresses, `--advertise-addr` must be specified so that the
correct address is chosen for inter-manager communication and overlay
networking.

It is also possible to specify a network interface to advertise that
interface's address; for example `--advertise-addr eth0:2377`.

Specifying a port is optional. If the value is a bare IP address or interface
name, the default port 2377 is used.

### <a name="data-path-addr"></a> Specify interface for data traffic (--data-path-addr)

The `--data-path-addr` flag specifies the address that global scope network
drivers will publish towards other nodes in order to reach the containers
running on this node. Using this parameter you can separate the container's
data traffic from the management traffic of the cluster.

If unspecified, the IP address or interface of the advertise address is used.

Setting `--data-path-addr` does not restrict which interfaces or source IP
addresses the VXLAN socket is bound to. Similar to `--advertise-addr`, the
purpose of this flag is to inform other members of the swarm about which
address to use for control plane traffic. To restrict access to the VXLAN port
of the node, use firewall rules.

### <a name="data-path-port"></a> Configure port number for data traffic (--data-path-port)

The `--data-path-port` flag allows you to configure the UDP port number to use
for data path traffic. The provided port number must be within the 1024 - 49151
range. If this flag isn't set, or if it's set to 0, the default port number
4789 is used. The data path port can only be configured when initializing the
swarm, and applies to all nodes that join the swarm. The following example
initializes a new Swarm, and configures the data path port to UDP port 7777;

```console
$ docker swarm init --data-path-port=7777
```

After the swarm is initialized, use the `docker info` command to verify that
the port is configured:

```console
$ docker info
<...>
ClusterID: 9vs5ygs0gguyyec4iqf2314c0
Managers: 1
Nodes: 1
Data Path Port: 7777
<...>
```

### <a name="default-addr-pool"></a> Specify default subnet pools (--default-addr-pool)

The `--default-addr-pool` flag specifies default subnet pools for global scope
networks. For example, to specify two address pools:

```console
$ docker swarm init \
  --default-addr-pool 30.30.0.0/16 \
  --default-addr-pool 40.40.0.0/16
```

Use the `--default-addr-pool-mask-length` flag to specify the default subnet
pools mask length for the subnet pools.

### <a name="max-snapshots"></a> Set limit for number of snapshots to keep (--max-snapshots)

This flag sets the number of old Raft snapshots to retain in addition to the
current Raft snapshots. By default, no old snapshots are retained. This option
may be used for debugging, or to store old snapshots of the swarm state for
disaster recovery purposes.

### <a name="snapshot-interval"></a> Configure Raft snapshot log interval (--snapshot-interval)

The `--snapshot-interval` flag specifies how many log entries to allow in
between Raft snapshots. Setting this to a high number will trigger snapshots
less frequently. Snapshots compact the Raft log and allow for more efficient
transfer of the state to new managers. However, there is a performance cost to
taking snapshots frequently.

### <a name="availability"></a> Configure the availability of a manager (--availability)

The `--availability` flag specifies the availability of the node at the time
the node joins a master. Possible availability values are `active`, `pause`, or
`drain`.

This flag is useful in certain situations. For example, a cluster may want to
have dedicated manager nodes that don't serve as worker nodes. You can do this
by passing `--availability=drain` to `docker swarm init`.

## Related commands

* [swarm ca](swarm_ca.md)
* [swarm join](swarm_join.md)
* [swarm join-token](swarm_join-token.md)
* [swarm leave](swarm_leave.md)
* [swarm unlock](swarm_unlock.md)
* [swarm unlock-key](swarm_unlock-key.md)
* [swarm update](swarm_update.md)
