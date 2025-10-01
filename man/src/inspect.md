This displays the low-level information on Docker object(s) (e.g. container, 
image, volume,network, node, service, or task) identified by name or ID. By default,
this will render all results in a JSON array. If the container and image have
the same name, this will return container JSON for unspecified type. If a format
is specified, the given template will be executed for each result.

# EXAMPLES

Get information about an image when image name conflicts with the container name,
e.g. both image and container are named rhel7:

    $ docker inspect --type=image rhel7
    [
    {
     "Id": "fe01a428b9d9de35d29531e9994157978e8c48fa693e1bf1d221dffbbb67b170",
     "Parent": "10acc31def5d6f249b548e01e8ffbaccfd61af0240c17315a7ad393d022c5ca2",
     ....
    }
    ]

## Getting information on a container

To get information on a container use its ID or instance name:

    $ docker inspect d2cc496561d6
    [{
    "Id": "d2cc496561d6d520cbc0236b4ba88c362c446a7619992123f11c809cded25b47",
    "Created": "2015-06-08T16:18:02.505155285Z",
    "Path": "bash",
    "Args": [],
    "State": {
        "Running": false,
        "Paused": false,
        "Restarting": false,
        "OOMKilled": false,
        "Dead": false,
        "Pid": 0,
        "ExitCode": 0,
        "Error": "",
        "StartedAt": "2015-06-08T16:18:03.643865954Z",
        "FinishedAt": "2015-06-08T16:57:06.448552862Z"
    },
    "Image": "ded7cd95e059788f2586a51c275a4f151653779d6a7f4dad77c2bd34601d94e4",
    "NetworkSettings": {
        "Networks": {
            "bridge": {
                "NetworkID": "7ea29fc1412292a2d7bba362f9253545fecdfa8ce9a6e37dd10ba8bee7129812",
                "EndpointID": "7587b82f0dada3656fda26588aee72630c6fab1536d36e394b2bfbcf898c971d",
                "Gateway": "172.17.0.1",
                "IPAddress": "172.17.0.2",
                "IPPrefixLen": 16,
                "IPv6Gateway": "",
                "GlobalIPv6Address": "",
                "GlobalIPv6PrefixLen": 0,
                "MacAddress": "02:42:ac:12:00:02"
            }
        },
        "Ports": {
            "80/tcp": [
                {
                    "HostIp": "0.0.0.0",
                    "HostPort": "8080"
                },
                {
                    "HostIp": "::",
                    "HostPort": "8080"
                }
            ]
        },
        "SandboxID": "6b4851d1903e16dd6a567bd526553a86664361f31036eaaa2f8454d6f4611f6f",
        "SandboxKey": "/var/run/docker/netns/6b4851d1903e"
    },
    "ResolvConfPath": "/var/lib/docker/containers/d2cc496561d6d520cbc0236b4ba88c362c446a7619992123f11c809cded25b47/resolv.conf",
    "HostnamePath": "/var/lib/docker/containers/d2cc496561d6d520cbc0236b4ba88c362c446a7619992123f11c809cded25b47/hostname",
    "HostsPath": "/var/lib/docker/containers/d2cc496561d6d520cbc0236b4ba88c362c446a7619992123f11c809cded25b47/hosts",
    "LogPath": "/var/lib/docker/containers/d2cc496561d6d520cbc0236b4ba88c362c446a7619992123f11c809cded25b47/d2cc496561d6d520cbc0236b4ba88c362c446a7619992123f11c809cded25b47-json.log",
    "Name": "/adoring_wozniak",
    "RestartCount": 0,
    "Driver": "overlay2",
    "MountLabel": "",
    "ProcessLabel": "",
    "Mounts": [
      {
        "Source": "/data",
        "Destination": "/data",
        "Mode": "ro,Z",
        "RW": false
        "Propagation": ""
      }
    ],
    "AppArmorProfile": "",
    "ExecIDs": null,
    "HostConfig": {
        "Binds": null,
        "ContainerIDFile": "",
        "Memory": 0,
        "MemorySwap": 0,
        "CpuShares": 0,
        "CpuPeriod": 0,
        "CpusetCpus": "",
        "CpusetMems": "",
        "CpuQuota": 0,
        "BlkioWeight": 0,
        "OomKillDisable": false,
        "Privileged": false,
        "PortBindings": {},
        "Links": null,
        "PublishAllPorts": false,
        "Dns": null,
        "DnsSearch": null,
        "DnsOptions": null,
        "ExtraHosts": null,
        "VolumesFrom": null,
        "Devices": [],
        "NetworkMode": "bridge",
        "IpcMode": "",
        "PidMode": "",
        "UTSMode": "",
        "CapAdd": null,
        "CapDrop": null,
        "RestartPolicy": {
            "Name": "no",
            "MaximumRetryCount": 0
        },
        "SecurityOpt": null,
        "ReadonlyRootfs": false,
        "Ulimits": null,
        "LogConfig": {
            "Type": "json-file",
            "Config": {}
        },
        "CgroupParent": ""
    },
    "GraphDriver": {
        "Data": {
            "LowerDir": "/var/lib/docker/overlay2/44b1d1f04db6b1b73a86f9a62678673bf5d16d9a6b62c13e859aa34a99cce5ea/diff:/var/lib/docker/overlay2/ef637181eb13e30e84b7382183364ed7fd7ff7be22d8bb87049e36b75fb89a86/diff:/var/lib/docker/overlay2/64fb0f850b1289cd09cbc3b077cab2c0f59a4f540c67f997b094fc3652b9b0d6/diff:/var/lib/docker/overlay2/68c4d1411addc2b2bd07e900ca3a059c9c5f9fa2607efd87d8d715a0080ed242/diff",
            "MergedDir": "/var/lib/docker/overlay2/c7846fe68c6f18247ab9b8672114dde9f506bc164081a895c465716eeb10f2bc/merged",
            "UpperDir": "/var/lib/docker/overlay2/c7846fe68c6f18247ab9b8672114dde9f506bc164081a895c465716eeb10f2bc/diff",
            "WorkDir": "/var/lib/docker/overlay2/c7846fe68c6f18247ab9b8672114dde9f506bc164081a895c465716eeb10f2bc/work"
        },
        "Name": "overlay2"
    },
    "Config": {
        "Hostname": "d2cc496561d6",
        "Domainname": "",
        "User": "",
        "AttachStdin": true,
        "AttachStdout": true,
        "AttachStderr": true,
        "ExposedPorts": null,
        "Tty": true,
        "OpenStdin": true,
        "StdinOnce": true,
        "Env": null,
        "Cmd": [
            "bash"
        ],
        "Image": "fedora",
        "Volumes": null,
        "VolumeDriver": "",
        "WorkingDir": "",
        "Entrypoint": null,
        "NetworkDisabled": false,
        "MacAddress": "",
        "OnBuild": null,
        "Labels": {},
        "Memory": 0,
        "MemorySwap": 0,
        "CpuShares": 0,
        "Cpuset": "",
        "StopSignal": "SIGTERM"
    }
    }
    ]
## Getting the IP address of a container instance

To get the IP address of a container use:

    $ docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' d2cc496561d6
    172.17.0.2

## Listing all port bindings

One can loop over arrays and maps in the results to produce simple text
output:

    $ docker inspect --format='{{range $p, $conf := .NetworkSettings.Ports}} \
      {{$p}} -> {{(index $conf 0).HostPort}} {{end}}' d2cc496561d6
      80/tcp -> 80

You can get more information about how to write a Go template from:
https://pkg.go.dev/text/template.

## Getting size information on a container

    $ docker inspect -s d2cc496561d6
    [
    {
    ....
    "SizeRw": 0,
    "SizeRootFs": 972,
    ....
    }
    ]

## Getting information on an image

Use an image's ID or name (e.g., repository/name[:tag]) to get information
about the image:

    docker inspect hello-world
    [
        {
            "Id": "sha256:54e66cc1dd1fcb1c3c58bd8017914dbed8701e2d8c74d9262e26bd9cc1642d31",
            "RepoTags": [
                "hello-world:latest"
            ],
            "RepoDigests": [
                "hello-world@sha256:54e66cc1dd1fcb1c3c58bd8017914dbed8701e2d8c74d9262e26bd9cc1642d31"
            ],
            "Parent": "",
            "Comment": "buildkit.dockerfile.v0",
            "Created": "2025-08-08T19:05:17Z",
            "DockerVersion": "",
            "Author": "",
            "Config": {
                "Env": [
                    "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
                ],
                "Cmd": [
                    "/hello"
                ],
                "WorkingDir": "/"
            },
            "Architecture": "arm64",
            "Variant": "v8",
            "Os": "linux",
            "Size": 16963,
            "RootFS": {
                "Type": "layers",
                "Layers": [
                    "sha256:50163a6b11927e67829dd6ba5d5ba2b52fae0a17adb18c1967e24c13a62bfffa"
                ]
            },
            "Metadata": {
                "LastTagTime": "2025-09-30T14:06:24.148634215Z"
            },
            "Descriptor": {
                "mediaType": "application/vnd.oci.image.index.v1+json",
                "digest": "sha256:54e66cc1dd1fcb1c3c58bd8017914dbed8701e2d8c74d9262e26bd9cc1642d31",
                "size": 12341
            }
        }
    ]
