# plugin push

<!---MARKER_GEN_START-->
Push a plugin to a registry

### Options

| Name                      | Type   | Default | Description        |
|:--------------------------|:-------|:--------|:-------------------|
| `--disable-content-trust` | `bool` | `true`  | Skip image signing |


<!---MARKER_GEN_END-->

## Description

After you have created a plugin using `docker plugin create` and the plugin is
ready for distribution, use `docker plugin push` to share your images to Docker
Hub or a self-hosted registry.

Registry credentials are managed by [docker login](login.md).

## Examples

The following example shows how to push a sample `user/plugin`.

```console
$ docker plugin ls

ID             NAME                    DESCRIPTION                  ENABLED
69553ca1d456   user/plugin:latest      A sample plugin for Docker   false

$ docker plugin push user/plugin
```

## Related commands

* [plugin create](plugin_create.md)
* [plugin disable](plugin_disable.md)
* [plugin enable](plugin_enable.md)
* [plugin inspect](plugin_inspect.md)
* [plugin install](plugin_install.md)
* [plugin ls](plugin_ls.md)
* [plugin rm](plugin_rm.md)
* [plugin set](plugin_set.md)
* [plugin upgrade](plugin_upgrade.md)
