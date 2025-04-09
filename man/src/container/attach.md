The **docker attach** command allows you to attach to a running container using
the container's ID or name, either to view its ongoing output or to control it
interactively.  You can attach to the same contained process multiple times
simultaneously, screen sharing style, or quickly view the progress of your
detached process.

To stop a container, use `CTRL-c`. This key sequence sends **SIGKILL** to the
container. You can detach from the container (and leave it running) using a
configurable key sequence. The default sequence is `CTRL-p CTRL-q`. You
configure the key sequence using the **--detach-keys** option or a configuration
file. See **config-json(5)** for documentation on using a configuration file.

It is forbidden to redirect the standard input of a **docker attach** command while
attaching to a TTY-enabled container (i.e., launched with `-i` and `-t`).

# EXAMPLES

## Attaching to a container

In this example the top command is run inside a container from an ubuntu image,
in detached mode, then attaches to it, and then terminates the container
with `CTRL-c`:

    $ docker run -d --name topdemo alpine top -b
    $ docker attach topdemo
    Mem: 2395856K used, 5638884K free, 2328K shrd, 61904K buff, 1524264K cached
    CPU:   0% usr   0% sys   0% nic  99% idle   0% io   0% irq   0% sirq
    Load average: 0.15 0.06 0.01 1/567 6
    PID  PPID USER     STAT   VSZ %VSZ CPU %CPU COMMAND
    1     0 root     R     1700   0%   3   0% top -b
    ^C

## Override the detach sequence

Use the **--detach-keys** option to override the Docker key sequence for detach.
This is useful if the Docker default sequence conflicts with key sequence you
use for other applications. There are two ways to define your own detach key
sequence, as a per-container override or as a configuration property on  your
entire configuration.

To override the sequence for an individual container, use the
**--detach-keys**=*key* flag with the **docker attach** command. The format of
the *key* is either a letter [a-Z], or the **ctrl**-*value*, where *value* is one
of the following:

* **a-z** (a single lowercase alpha character )
* **@** (at sign)
* **[** (left bracket)
* **\\\\** (two backward slashes)
* **_** (underscore)
* **^** (caret)

These **a**, **ctrl-a**, **X**, or **ctrl-\\** values are all examples of valid key
sequences. To configure a different configuration default key sequence for all
containers, see **docker(1)**.

