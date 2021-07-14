---
title: "builder prune"
description: "The builder prune operations with various inputs and usage"
keywords: "builder, docker, prune"
redirect_from:
- /reference/commandline/builder_prune/
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# Description

Working with Docker, docker have various unused objects hanging around from previous activities, so docker
takes a conservative approach to clean up unused objects, part of the object which we would be talking about is the builder cache
these object are stored when building dockerfiles, there're generally not removed unless you explicitly ask Docker to do so. This
can cause Docker to use extra disk space. Docker provides a prune command `docker builder prune` or `docker buildx prune` to clean
up multiple unused build cache at once. This topic shows how to use these prune command and it different inputs to clean build
cache.

When we execute the `$ docker builder prune` command, it makes an API call to the Docker daemon, and the daemon searches for all
unused build cache objects on that host and removes those objects from the system.

The goal of creating cache is to speed up future builds.
Removing unused build cache is very important in docker so as to build effective dockerfiles, pruning removes dead and unused
build cache allowing room for new and useful cache. It also defers bad build cache and promotes the builds to work faster and
effective.
So docker try to remove the cache that is less likely to be used in future builds to keep builds as fast as possible while maintaining cache size under control.

**Note**
>Pruning does not works if the Total is below what you specified as keep storage option to prune. It needs to go above, we are
>going to see how to use the `--keep-storage` option as part of our examples in this topic.

# builder prune

```markdown
Usage:  docker builder prune [OPTIONS]

Remove build cache

Options:
  -a, --all                  Remove all unused build cache, not just dangling ones
      --filter filter        Provide filter values (e.g. 'until=24h')
  -f, --force                Do not prompt for confirmation
      --keep-storage bytes   Amount of disk space to keep for cache
```

# Usage

To see how much cache you are using currently you can use `docker system df` to get a total view for what’s being in use including
containers, images, and build cache.
Our build caches records can be inspected with the following command: `docker system df -v`
(it shows image, container and volume caches)
or with buildx command via: `buildx du --verbose`
with a bit different output.

```bash
$ docker system df -v

Build cache usage: 2.97GB

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
y6xp6ur6a7ku        regular             265B                2 weeks ago         2 weeks ago         1                   false
ace0eda3e3be        regular             5.57MB              2 weeks ago         2 weeks ago         2                   false
db9945da3e79        regular             4.68kB              2 weeks ago         2 weeks ago         2                   false
6a3365937e92        regular             0B                  2 weeks ago         2 weeks ago         2                   false
271d635f4277        regular             49.9MB              2 weeks ago         2 weeks ago         2                   false
2e86a56bd8aa        regular             138B                2 weeks ago         2 weeks ago         3                   false
rungovuto78m        regular             311B                2 weeks ago         2 weeks ago         1                   false
nhqetzusaoxp        source.local        695B                2 weeks ago         2 weeks ago         1                   false
7sg0fy1pmvtt        regular             0B                  2 weeks ago         2 weeks ago         1                   false
gi8wypsigtjm        source.local        340B                2 weeks ago         2 weeks ago         1                   false
cd21hgno3hax        regular             2.42MB              2 weeks ago         2 weeks ago         3                   true
36bctagchywz        regular             540MB               2 weeks ago         2 weeks ago         3                   false
l1wnrjeqm9pa        regular             81MB                2 weeks ago         2 weeks ago         3                   false
1jr4qt5wqgjl        regular             0B                  2 weeks ago         2 weeks ago         3                   false
9780f6d83e45        regular             114MB               2 weeks ago         2 weeks ago         4                   false
5173011923d0        regular             16.5MB              2 weeks ago         2 weeks ago         4                   false
4bb57adf9037        regular             17.5MB              2 weeks ago         2 weeks ago         4                   false
d5d618196ec3        regular             146MB               1 weeks ago         1 weeks ago         4                   false
1a893709dfe5        regular             32.9MB              2 minutes ago       2 minutes ago       4                   false
a5998494261f        regular             345MB               7 minutes ago       7 minutes ago       4                   false
```

The [docker builder prune](commandline/builder_prune.md) command line remove all build cache
that are dangling without specifying any of the prune flag [OPTIONS].

Before using the `docker builder prune` command, this is what the cache record looks like with the `docker system df -v` command:

```bash
$ docker system df -v

Build cache usage: 647MB

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
y6xp6ur6a7ku        regular             265MB               2 weeks ago         2 weeks ago         1                   false
ace0eda3e3be        regular             5.57MB              2 weeks ago         2 weeks ago         0                   false
db9945da3e79        regular             254MB               2 weeks ago         2 weeks ago         1                   false
6a3365937e92        regular             15.84MB             2 weeks ago         2 weeks ago         0                   false
271d635f4277        regular             49.9MB              2 weeks ago         2 weeks ago         2                   false
2e86a56bd8aa        regular             38MB                2 weeks ago         2 weeks ago         3                   false
rungovuto78m        regular             311B                2 weeks ago         2 weeks ago         0                   false
nhqetzusaoxp        source.local        695B                2 weeks ago         2 weeks ago         1                   false
7sg0fy1pmvtt        regular             10MB                2 weeks ago         2 weeks ago         0                   false
gi8wypsigtjm        source.local        340B                2 weeks ago         2 weeks ago         1                   false
```
This was the record after using the `docker builder prune` command:

```bash
$ docker system df -v

Build cache usage: 647MB

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
y6xp6ur6a7ku        regular             265MB               2 weeks ago         2 weeks ago         1                   false
db9945da3e79        regular             254MB               2 weeks ago         2 weeks ago         1                   false
271d635f4277        regular             49.9MB              2 weeks ago         2 weeks ago         2                   false
2e86a56bd8aa        regular             38MB                2 weeks ago         2 weeks ago         3                   false
nhqetzusaoxp        source.local        695B                2 weeks ago         2 weeks ago         1                   false
gi8wypsigtjm        source.local        340B                2 weeks ago         2 weeks ago         1                   false
```
As noticed, some dangeling cache are deleted from the record based on the USAGE Status. Does cache record showing 0 usage are not
being used and therefore are been deleted, also if a cache is been shared by two or more images i.e SHARED = true, the cache can 
not be deleted, but we have all the cache in false (they are not being shared). Cache can not be deleted if `USAGE = 0` 
and `SHARED = true` same applies to `USAGE = 1` and `SHARED = false` both of the cache attribute needs to correlate.
i.e `SHARED = false` and `USAGE = 0`.


# Examples

The [docker builder prune --help](commandline/builder_prune.md) shows all the flag / options that can be used for builder prune.
This is what the output looks like:

```bash
$ docker builder prune --help

Usage:  docker builder prune

Remove build cache

Options:
  -a, --all                  Remove all unused build cache, not just dangling ones
      --filter filter        Provide filter values (e.g. 'until=24h')
  -f, --force                Do not prompt for confirmation
      --keep-storage bytes   Amount of disk space to keep for cache
```

# Build Prune With (--force) flag

First of all, the `--force` `-f` flag in [docker builder prune](commandline/builder_prune.md) helps to bypass
the authentication that pops up when we use `docker builder prune` and any of its options. it can be used with other
options, applications and usage examples of `--force` are shown with other options below.

> **Note**
> This feature dose not allow you to force remove a cache that is in use.
> it only bypass authentications.


# Build Prune With (--all, -a) flag 

Builder prune has a flag for removing all unused build cache, not just dangling caches, Discription of this flag is below.
First we check for builder caches:

```bash
$ docker system df -v

Build cache usage: 941.3MB

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
a5998494261f        regular             345MB               2 weeks ago         2 weeks ago         4                   false
bdd3e2e51daf        regular             38.2MB              2 weeks ago         2 weeks ago         0                   false
zii5dwynmvgf        regular             3.56kB              2 weeks ago         2 weeks ago         2                   false
p04j8skx2cio        regular             4.89kB              2 weeks ago         2 weeks ago         2                   false
aph1w8hfh3u5        regular             15.4MB              2 weeks ago         2 weeks ago         1                   false
nj9l7si31hw7        regular             540MB               2 weeks ago         2 weeks ago         1                   false
2s1t824ku5g7        regular             2.42MB              2 weeks ago         2 weeks ago         0                   false
qb9txhrba7vx        regular             0B                  2 weeks ago         2 weeks ago         1                   false
nlaulj32zp6e        regular             4.89kB              2 weeks ago         2 weeks ago         3                   false
kyogwgykftyr        regular             0B                  2 weeks ago         2 weeks ago         0                   false
```
Now we use the `--all, -a` flag.

```bash
$ docker builder prune --all

# A warning notification pops up, that you would need to answer yes or no
# y answers yes to remove all build cache.
# n answers no to decline the removal of build cache.

WARNING! This will remove all build cache. Are you sure you want to continue? [y/N] y
# y was selected i.e delete all cache.

Deleted build cache objects:
bdd3e2e51dafnchruxumyash1
a5998494261fqksutu2630zkk
zii5dwynmvgfdlrdw7vkqup60
p04j8skx2cio7hsp60k1b7adm
aph1w8hfh3u5r6wxqngb45989
nj9l7si31hw7ih17y44qe412z
2s1t824ku5g7kthe8ecu6k3xe
qb9txhrba7vxxm9vo7r0p490p
nlaulj32zp6e5zgwl7z95cmk5
kyogwgykftyrmgvbd3wl3hg9y

Total reclaimed space: 941.3MB
```
## Using -force, -f flag with --all flag

This is an example that shows what an -f flag does in a `docker builder prune --all -f` or `docker builder prune --all --force`
command

```bash
$ docker system df -v # to check for build cache

Build cache usage: 112.78MB

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
d0fe97fa8b8c        regular             69.2MB              2 weeks ago         13 days ago         5                   false
4d6b84966400        regular             41.3MB              2 weeks ago         13 days ago         1                   false
bb186d194dec        regular             2.28MB              2 weeks ago         13 days ago         2                   false

$ docker builder prune --all -f #to remove all build cache with the force flag.

# NOTE: No pop up showed up to ask if you approve the removal of the build cache.

Deleted build cache objects:
d0fe97fa8b8cq19z9hqs5tphe
4d6b84966400179cjn6dgqtaz
bb186d194decu5qrg0jjmw6tp

Total reclaimed space: 112.78MB
```

# Build Prune With (--filter) flag 

The filter flag for `builder prune` is used to `filter` what type of filter you want to remove based on the usage time 
i.e the last time it was used, below we will show how the `--filter` flag is been used in our builder prune command.
Command: `docker builder prune --filter unused-for=24h` or `docker builder prune --filter until=24h`

What the command simply say is that remove every builder cache that has not been used for the past 24 hours or
that are not within 24 hours range.

**Note**
> Setting `unused-for` or `until` should be in Hours

```bash
$ docker system df -v ## to check the builder cache

Build cache usage: 652.98MB

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
e76a5m45h6pw        regular             74.5MB              14 hours ago        14 hours ago        1                   false
xlpunf7hd5yh        regular             36.1MB              17 hours ago        17 hours ago        1                   false
rx58kxz9kmlg        regular             540MB               22 hours ago        22 hours ago        1                   false
08unho1rnguz        regular             2.38MB              13 days ago         13 days ago         1                   false
i8nfk2fvpyyi        regular             0B                  2 weeks ago         13 days ago         2                   false

$ docker builder prune --filter until=24h # to remove builder cache with a filter or 24 hours

WARNING! This will remove all dangling build cache. Are you sure you want to continue? [y/N] y

# y (yes) was selected.

Deleted build cache objects:
08unho1rnguz179cjn6dgqtaz
i8nfk2fvpyyiu5qrg0jjmw6tp

Total reclaimed space: 2.38MB

# Two builder cache were deleted, the two were used 13 days ago, but we specify that we want to delete every build
# cache that has not been used for the last 24 hours.
```

### Example of --all and --filter flag combination.
```bash
$ docker system df -v ## to check the builder cache

Build cache usage: 336.28MB

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
rkze7nhuwsrh        regular             36.1MB              2 weeks ago         12 hours ago        1                   false
qemxnvg1n1o8        regular             36.1MB              2 weeks ago         2 weeks ago         2                   false
8l4s9m2j2z4b        regular             74.5MB              2 weeks ago         2 weeks ago         0                   false
d0fe97fa8b8c        regular             69.2MB              2 weeks ago         10 days ago         1                   false
4d6b84966400        regular             41.3MB              2 weeks ago         13 days ago         0                   false
bb186d194dec        regular             2.28MB              2 weeks ago         17 hours ago        1                   false
216461ee9eb5        regular             76.8MB              2 weeks ago         13 days ago         0                   false

$ docker builder prune --all --filter unused-for = 24h

WARNING! This will remove all dangling build cache. Are you sure you want to continue? [y/N] y
#y selected.

Deleted build cache objects:

8l4s9m2j2z4biu5qrg0jjmw6tp
4d6b84966400yyqyguxrhrdv8k
216461ee9eb5xm9vo7r0p490pg

Total reclaimed space: 38.38MB

# only build cache that are unused and are last used for more than 24 hours were deleted.
```

## Using -f, --force flag with --filter.
By now we know what the `-f` or `--force` flag does, in this example we are going to see how to 
implement the `-f` or `--force` flag with our `--filter` flag.

```bash
$ docker system df -v # to check for builder cache

Build cache usage: 8.625KB

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
hjjsk55wnswi        regular             0B                  2 weeks ago         13 days ago         6                   false
e6vrmmkb82mm        source.local        0B                  2 weeks ago         13 days ago         19                  false
g7hi043vg9ml        source.local        265B                3 days ago          2 hours ago         13                  false
8mtuu862jx59        regular             3.5kB               13 days ago         13 days ago         1                   false
dalvi76v52o2        regular             4.86kB              11 days ago         54 minutes ago      1                   false

$ docker builder prune --filter unused-for=24h --force

# NOTE: No pop up showed up to ask if you approve the removal of the build cache.
Deleted build cache objects:
hjjsk55wnswizmxpf551sp2qh
e6vrmmkb82mmffmls87xe41k8
8mtuu862jx59yyqyguxrhrdv8


Total reclaimed space: 3.5kb
```

# Build Prune With (--keep-storage) flag 

Docker as a way to manage the storage space that build cache can consume. The `--keep-storage` command helps
us to manage the amount of storage space that our build cache can have.

**Note** 
>If build cache as been asigned to only have 1GB of storage, i.e it can only store upto 1GB amount of build caches 
>and if the storage is full, it can no longer store more build caches therefore any other build will not be stored.
>i.e when the prune, it deletes the ones that are unused until it gets to 1GB.

Usage is below:

```bash 
$ docker builder prune --keep-storage 512MB

WARNING! This will remove all dangling build cache. Are you sure you want to continue? [y/N] y

# y (yes) was selected.

Total reclaimed space: 0B

# now let use prune our build cache

$ docker system df -v # to check for builder cache

Build cache usage: 624.5MB

CACHE ID            CACHE TYPE          SIZE                CREATED             LAST USED           USAGE               SHARED
nhqetzusaoxp        source.local        695B                2 weeks ago         2 weeks ago         0                   false
7sg0fy1pmvtt        regular             0B                  2 weeks ago         2 weeks ago         0                   false
gi8wypsigtjm        source.local        340B                2 weeks ago         2 weeks ago         1                   false
cd21hgno3hax        regular             2.42MB              2 weeks ago         2 weeks ago         3                   true
36bctagchywz        regular             540MB               2 weeks ago         2 weeks ago         0                   false
l1wnrjeqm9pa        regular             81MB                2 weeks ago         2 weeks ago         3                   false
1jr4qt5wqgjl        regular             0B                  2 weeks ago         2 weeks ago         3                   false
bdd3e2e51daf        regular             38.2MB              2 weeks ago         2 weeks ago         0                   false

$ docker builder prune --all # Remove all unused cache till its under 512MB(Keep Storage limit)

WARNING! This will remove all dangling build cache. Are you sure you want to continue? [y/N] y

nhqetzusaoxpxm9vo7r0p490p
7sg0fy1pmvtt5zgwl7z95cmk5
36bctagchywzmgvbd3wl3hg9y

# we have 4 unused cache there, but he removed 3 unused cache, because it has removed the amount of cache down to what we needed  
# 512MB keep storage, and based on our filter.

Total reclaimed space: 540.70MB
```
## Using -f, --force flag with --keep-storage.

In this example we are going to see how to 
implement the `-f` or `--force` flag with our `--keep-storage` flag.

```bash
$ docker builder prune --keep-storage 5gb -f

# NOTE: No pop up showed up to ask if you approve the removal of the build cache.
Total reclaimed space: 0B
```
[Garbage collection](commandline/garbagecollection_config.md) is done in the pruning process, its an ordered list of prune
operations, click the link to know more and see how to configure garbage collection. 