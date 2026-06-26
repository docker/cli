The **docker container update** command dynamically updates container configuration.
You can use this command to prevent containers from consuming too many 
resources from their Docker host.  With a single command, you can place 
limits on a single container or on many. To specify more than one container,
provide space-separated list of container names or IDs.

# OPTIONS

## memory

Memory limit (format: `<number>[<unit>]`, where unit = b, k, m or g)

Note that the memory should be smaller than the already set swap memory limit.
If you want update a memory limit bigger than the already set swap memory limit,
you should update swap memory limit at the same time. If you don't set swap memory 
limit on docker create/run but only memory limit, the swap memory is double
the memory limit.

# EXAMPLES

The following sections illustrate ways to use this command.

### Update a container's cpu-shares

To limit a container's cpu-shares to 512, first identify the container
name or ID. You can use **docker ps** to find these values. You can also
use the ID returned from the **docker run** command.  Then, do the following:

```console
$ docker container update --cpu-shares 512 abebf7571666
```

### Update a container with cpu-shares and memory

To update multiple resource configurations for multiple containers:

```console
$ docker container update --cpu-shares 512 -m 300M abebf7571666 hopeful_morse
```

### Update a container's restart policy

You can change a container's restart policy on a running container. The new
restart policy takes effect instantly after you run `docker container update` on a
container.

To update restart policy for one or more containers:

```console
$ docker container update --restart=on-failure:3 abebf7571666 hopeful_morse
```

Note that if the container is started with "--rm" flag, you cannot update the restart
policy for it. The `AutoRemove` and `RestartPolicy` are mutually exclusive for the
container.
