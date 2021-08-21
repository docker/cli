The `docker kill` subcommand kills one or more containers. The main process
inside the container is sent `SIGKILL` signal (default), or the signal that is
specified with the `--signal` option. You can reference a container by its
ID, ID-prefix, or name.

The `--signal` flag sets the system call signal that is sent to the container.
This signal can be a signal name in the format `SIG<NAME>`, for instance `SIGINT`,
or an unsigned number that matches a position in the kernel's syscall table,
for instance `2`.

While the default (`SIGKILL`) signal will terminate the container, the signal
set through `--signal` may be non-terminal, depending on the container's main
process. For example, the `SIGHUP` signal in most cases will be non-terminal,
and the container will continue running after receiving the signal.

> **Note**
>
> `ENTRYPOINT` and `CMD` in the *shell* form run as a child process of
> `/bin/sh -c`, which does not pass signals. This means that the executable is
> not the container’s PID 1 and does not receive Unix signals.
