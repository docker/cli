Removes (and un-tags) one or more images from the host node. If an image has
multiple tags, using this command with the tag as a parameter only removes the
tag. If the tag is the only one for the image, both the image and the tag are
removed.

This does not remove images from a registry. You cannot remove an image of a
running container unless you use the **-f** option. To see all images on a host
use the **docker image ls** command.

# EXAMPLES

## Removing an image

Here is an example of removing an image:

    docker image rm fedora/httpd
