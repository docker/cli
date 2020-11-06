Use `docker image push` to share your images to the [Docker Hub](https://hub.docker.com)
registry or to a self-hosted one.

Refer to **docker-image-tag(1)** for more information about valid image and tag names.

Killing the **docker image push** process, for example by pressing **CTRL-c** while it
is running in a terminal, terminates the push operation.

Registry credentials are managed by **docker-login(1)**.

# EXAMPLES

## Pushing a new image to a registry

First save the new image by finding the container ID (using **docker container ls**)
and then committing it to a new image name.  Note that only a-z0-9-_. are
allowed when naming images:

    # docker container commit c16378f943fe rhel-httpd

Now, push the image to the registry using the image ID. In this example the
registry is on host named `registry-host` and listening on port `5000`. To do
this, tag the image with the host name or IP address, and the port of the
registry:

    # docker image tag rhel-httpd registry-host:5000/myadmin/rhel-httpd:latest
    # docker image push registry-host:5000/myadmin/rhel-httpd:latest

Check that this worked by running:

    # docker image ls

You should see both `rhel-httpd` and `registry-host:5000/myadmin/rhel-httpd`
listed.

### Push all tags of an image

Use the `-a` (or `--all-tags`) option to push To push all tags of a local image.

The following example creates multiple tags for an image, and pushes all those
tags to Docker Hub.

    $ docker image tag myimage registry-host:5000/myname/myimage:latest
    $ docker image tag myimage registry-host:5000/myname/myimage:v1.0.1
    $ docker image tag myimage registry-host:5000/myname/myimage:v1.0
    $ docker image tag myimage registry-host:5000/myname/myimage:v1

The image is now tagged under multiple names:

    $ docker image ls
    
    REPOSITORY                          TAG        IMAGE ID       CREATED      SIZE
    myimage                             latest     6d5fcfe5ff17   2 hours ago  1.22MB
    registry-host:5000/myname/myimage   latest     6d5fcfe5ff17   2 hours ago  1.22MB
    registry-host:5000/myname/myimage   v1         6d5fcfe5ff17   2 hours ago  1.22MB
    registry-host:5000/myname/myimage   v1.0       6d5fcfe5ff17   2 hours ago  1.22MB
    registry-host:5000/myname/myimage   v1.0.1     6d5fcfe5ff17   2 hours ago  1.22MB

When pushing with the `--all-tags` option, all tags of the `registry-host:5000/myname/myimage`
image are pushed:


    $ docker image push --all-tags registry-host:5000/myname/myimage
    
    The push refers to repository [registry-host:5000/myname/myimage]
    195be5f8be1d: Pushed
    latest: digest: sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6 size: 4527
    195be5f8be1d: Layer already exists
    v1: digest: sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6 size: 4527
    195be5f8be1d: Layer already exists
    v1.0: digest: sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6 size: 4527
    195be5f8be1d: Layer already exists
    v1.0.1: digest: sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6 size: 4527
