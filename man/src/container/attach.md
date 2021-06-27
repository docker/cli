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
attaching to a tty-enabled container (i.e.: launched with `-t`).

# Override the detach sequence

If you want, you can configure an override the Docker key sequence for detach.
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

# EXAMPLES

## Attaching to a container

In this example the top command is run inside a container, from an image called
fedora, in detached mode. The ID from the container is passed into the **docker
attach** command:

    $ ID=$(sudo docker run -d ubuntu:20.04 /usr/bin/top -b)
    $ sudo docker attach $ID
    top - 00:07:01 up  4:54,  0 users,  load average: 0.83, 0.91, 0.82
    Tasks:   1 total,   1 running,   0 sleeping,   0 stopped,   0 zombie
    %Cpu(s):  2.3 us,  1.6 sy,  0.0 ni, 95.9 id,  0.0 wa,  0.1 hi,  0.1 si,  0.0 st
    MiB Mem :  15846.2 total,   5729.2 free,   2592.5 used,   7524.4 buff/cache
    MiB Swap:  16384.0 total,  16384.0 free,      0.0 used.  12097.3 avail Mem 
    
        PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
          1 root      20   0    5976   3256   2828 R   0.0   0.0   0:00.04 top
    
    top - 00:07:04 up  4:54,  0 users,  load average: 0.76, 0.89, 0.81
    Tasks:   1 total,   1 running,   0 sleeping,   0 stopped,   0 zombie
    %Cpu(s):  2.0 us,  1.4 sy,  0.0 ni, 96.5 id,  0.0 wa,  0.1 hi,  0.0 si,  0.0 st
    MiB Mem :  15846.2 total,   5727.5 free,   2594.4 used,   7524.3 buff/cache
    MiB Swap:  16384.0 total,  16384.0 free,      0.0 used.  12095.6 avail Mem 
    
        PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
      1 root      20   0    5976   3256   2828 R   0.0   0.0   0:00.04 top
