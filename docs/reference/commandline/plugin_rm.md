---
title: "plugin rm"
description: "the plugin rm command description and usage"
keywords: "plugin, rm"
---

# plugin rm

```markdown
Usage:  docker plugin rm [OPTIONS] PLUGIN [PLUGIN...]

Remove one or more plugins

Aliases:
  rm, remove

Options:
      -f, --force  Force the removal of an active plugin
          --help   Print usage
```

## Description

Removes a plugin. You cannot remove a plugin if it is enabled, you must disable
a plugin using the [`docker plugin disable`](plugin_disable.md) before removing
it (or use --force, use of force is not recommended, since it can affect
functioning of running containers using the plugin).

## Examples

The following example disables and removes the `sample-volume-plugin:latest`
plugin:

```console
$ docker plugin disable tiborvass/sample-volume-plugin

tiborvass/sample-volume-plugin

$ docker plugin rm tiborvass/sample-volume-plugin:latest

tiborvass/sample-volume-plugin
```

## Related commands

* [plugin create](plugin_create.md)
* [plugin disable](plugin_disable.md)
* [plugin enable](plugin_enable.md)
* [plugin inspect](plugin_inspect.md)
* [plugin install](plugin_install.md)
* [plugin ls](plugin_ls.md)
* [plugin push](plugin_push.md)
* [plugin set](plugin_set.md)
* [plugin upgrade](plugin_upgrade.md)
