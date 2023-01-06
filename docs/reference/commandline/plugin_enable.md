# plugin enable

<!---MARKER_GEN_START-->
Enable a plugin

### Options

| Name        | Type  | Default | Description                      |
|:------------|:------|:--------|:---------------------------------|
| `--timeout` | `int` | `30`    | HTTP client timeout (in seconds) |


<!---MARKER_GEN_END-->

## Description

Enables a plugin. The plugin must be installed before it can be enabled,
see [`docker plugin install`](plugin_install.md).

## Examples

The following example shows that the `sample-volume-plugin` plugin is installed,
but disabled:

```console
$ docker plugin ls

ID            NAME                                    DESCRIPTION                ENABLED
69553ca1d123  tiborvass/sample-volume-plugin:latest   A test plugin for Docker   false
```

To enable the plugin, use the following command:

```console
$ docker plugin enable tiborvass/sample-volume-plugin

tiborvass/sample-volume-plugin

$ docker plugin ls

ID            NAME                                    DESCRIPTION                ENABLED
69553ca1d123  tiborvass/sample-volume-plugin:latest   A test plugin for Docker   true
```

## Related commands

* [plugin create](plugin_create.md)
* [plugin disable](plugin_disable.md)
* [plugin inspect](plugin_inspect.md)
* [plugin install](plugin_install.md)
* [plugin ls](plugin_ls.md)
* [plugin push](plugin_push.md)
* [plugin rm](plugin_rm.md)
* [plugin set](plugin_set.md)
* [plugin upgrade](plugin_upgrade.md)
