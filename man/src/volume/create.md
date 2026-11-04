Creates a new volume that containers can consume and store data in. If a name
is not specified, Docker generates a random name. You create a volume and then
configure the container to use it, for example:

    $ docker volume create hello
    hello
    $ docker run -d -v hello:/world busybox ls /world

The mount is created inside the container's `/world` directory. Docker does
not support relative paths for mount points inside the container.

Multiple containers can use the same volume. This is useful if two containers
need access to shared data. For example, if one container writes and the other
reads the data.

Volume names must be unique among drivers. Attempting to create two volumes
with the same name on different drivers results in an error. If you specify a
volume name already in use on the current driver, Docker reuses the existing
volume and does not return an error.

## Driver specific options

Some volume drivers may take options to customize the volume creation. Use the
`-o` or `--opt` flags to pass driver options:

    $ docker volume create --driver fake --opt tardis=blue --opt timey=wimey

These options are passed directly to the volume driver. Options for different
volume drivers may do different things (or nothing at all).

The built-in `local` driver on Windows does not support any options.

The built-in `local` driver on Linux accepts options similar to the linux
`mount` command:

    $ docker volume create --driver local --opt type=tmpfs --opt device=tmpfs --opt o=size=100m,uid=1000

Another example:

    $ docker volume create --driver local --opt type=btrfs --opt device=/dev/sda2
