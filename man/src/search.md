Search Docker Hub for images that match the specified `TERM`. The table
of images returned displays the name, description (truncated by default), number
of stars awarded, whether the image is official, and whether it is automated.

## Filter

   Filter output based on these conditions:
   - stars=<numberOfStar>
   - is-automated=(true|false) (deprecated)
   - is-official=(true|false)

# EXAMPLES

## Search Docker Hub for ranked images

Search a registry for the term 'fedora' and only display those images
ranked 3 or higher:

    $ docker search --filter=stars=3 fedora
    NAME      DESCRIPTION                        STARS     OFFICIAL
    fedora    Official Docker builds of Fedora   1150      [OK]
