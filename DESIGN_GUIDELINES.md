# Docker CLI design guidelines

This document provides guidelines to develop new features and enhancements
for the Docker CLI.


This is not an exhaustive set of rules, so in case of doubt, it may be good
to discuss with a maintainer. This document is meant to be a living document;
if you see something out of date or missing, pull requests are welcome.

This document intends to;

- Provide a consistent, predictable UX for users of the Docker CLI
- Assist contributors and maintainers when reviewing changes, providing them
  guidelines to verify the design.

## General acceptance criteria

### Problem description

Features should address a problem or use-case. When contributing a new feature,
describe the use-case or problem that it is addressing. Having actual examples
not only helps to verify if the design matches the expectations, but can also
assist other participants to verify if the proposed solution is the _only_
(or "best") solution.

### Docker Compose file

Any feature added should usually also be added to the docker-compose schema;

- Propose new additions to the compose spec in the [Compose Spec](https://github.com/compose-spec/compose-spec)
  repository.
- Update schema files [in this repository](https://github.com/docker/cli/tree/master/cli/compose/schema)

### Documentation

Any feature added should be accompanied by at least:

- A mention in the corresponding reference documentation.
- An example in the reference documentation; the example
  should make clear _why_ a feature should be used (clear use-case)

For larger features, additional documentation may be needed in the main [documentation
 repository](https://github.com/docker/docker.github.io). The documentation team
can help with writing that documentation, but technical assistance from contributors
is generally appreciated.

### Completion scripts

New commands and flags should be added to the completion scripts. Help can be
provided in updating those scripts; it's acceptable to update completion scripts
in a follow-up pull requests, but a tracking issue must be created in that case.

- Bash completion (required)
- PowerShell (optional)
- Fish (optional)
- Zsh (optional)


## Technical / design debt

The Docker CLI evolved over time, which also means that the design inherited
some design-choices from the past that may not have been the best choices (in
hindsight), but cannot be changed without introducing breaking changes.

This section describes some of these behaviors.

### Legacy top level commands

Historically, the Docker CLI had a limited set of commands, to manage images
and containers (`docker run`, `docker pull`, `docker push`). With the introduction
of other type of objects (volumes, networks, services, plugins), this pattern
did not scale.

No new top-level commands should be added; top-level commands are reserved for
management commands going forward.

> **Note**: some less-frequently used legacy top-level commands can be hidden by
> setting the `DOCKER_HIDE_LEGACY_COMMANDS` environment variable. Setting this
> variable _hides_ the commands, but they will still remain active for backward
> compatibility.

### Exit code for filtered results

When filtering results, and no results were found, the CLI produces a zero (success)
exit code. Reason for this is that the action was _successful_, but happened to
not produce any matching results.

For example;

```bash
docker container ls --filter status=exited

CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES

echo $?
0
```

```bash
ls *.bla 2> /dev/null || echo "no such thing"
no such thing
```


Producing a zero exit code can complicate using these commands in scripting
situations, but has been considered too much of a breaking change to change (See
[#27657](https://github.com/moby/moby/issues/27657#issuecomment-258271259) and
[#28951](https://github.com/moby/moby/issues/28951)).

### Single-value flags can be specified multiple times

The Docker CLI accepts flags to be set multiple times, even if an option that's
set through that flag accepts a single value. If a single-value flag is set
multiple times, no error is produced, and latter values override prior values.

For example, the following command runs successfully;

```bash
container create --name one --name two --name three busybox
c872b39344646e86ef7d83f846e47f0cb92abc200938187f87052737026433d1

echo $?
0
```

And creates a container named `three`:

```bash
docker container inspect --format '{{.Name}}' c872b39344646e86ef7d83f846e47f0cb92abc200938187f87052737026433d1
/three
```

> **Note**: it may be worth revisiting this situation, and only keep this
> behavior for existing flags (for backward compatibility), and enforce single-
> value flags to be specified only once going forward.

### Passing input from `stdin`

There is some inconsistency in notations used to pass input from `stdin`.

Some commands accept a path as positional argument, and `-` to accept input from
`stdin`:

```
Usage:	docker build [OPTIONS] PATH | URL | -
Usage:	docker config create [OPTIONS] CONFIG file|-
Usage:	docker import [OPTIONS] file|URL|- [REPOSITORY[:TAG]]
```

Whereas (e.g.) `docker load` _default_ to using `stdin/stdout`, and require `-i` /
`--input` / `-o` / `--output` to be passed to use a file instead.

```
Usage:	docker save [OPTIONS] IMAGE [IMAGE...]
Usage:	docker load [OPTIONS]
Usage:	docker export [OPTIONS] CONTAINER
```

Which allows passing

```bash
echo $SOME_ENV_VAR | docker config create myconfig -

printf 'my configuration' | docker config create myconfig -

docker config create myconfig - <<< "my configuration"

docker config create myconfig - <<-'EOF'
line 1
line 2
EOF
```

### Sending output to `stdout`



> **Todo**: we need to describe the canonical approach going forward.

## General guidelines

### Keep it simple

Before contributing a feature, consider if the feature can be addressed through
other means. Try to adhere to [the Linux principle](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well);
"do one thing, and do it well". If a use-case can be addressed combining
commands (e.g. `docker container ls -q | xargs docker container inspect`).


### Do not prematurely optimize usability

### Do not use shorthand (single-letter) flags

Shorthand, single-letter flags can easily become ambiguous (for example, `-f`
can be either a shorthand for `--format` or for `--force`).

For this reason, shorthand flags must be reserved for frequently used options
only, and only if there is a need. As with all changes, it is easier to add
a shorthand option later, than to remove an option once added.

Standard flags/options are an exception to this rule, for example, list-commands
that have a `--format` option should generally also get a `-f` shorthand.

### Avoid microformats






### Configuration, and order of preference

1. Flag
2. Environment variable
3. Configuration



- not everything should be configurable. no direct need? don't do it


## Linux principle, and "chainable"


## Naming conventions

- Avoid product names: prefer "generic"
- Describe what it _does_, not how the product is named



## API and feature compatibility

The Docker CLI should be compatible with older versions of the API. If a feature
depends on a specific API version, or (for example) requires an orchestrator to
be enabled, then the feature should be hidden if those conditions are not met.

For example, the following code defines a `--foo` flag that requires API version
`1.99`. The flag will be hidden if the Docker CLI connects with a daemon that
does not support this version of the API.

```go
flags.BoolVar(&options.foo, "foo", false, "Foo enables foo on a container")
flags.SetAnnotation("foo", "version", []string{"1.99"})
```

The example below shows a `foo` command that requires a daemon with API version
1.99 and experimental features enabled. In this example, `foo` is a top-level
command, and all sub-commands will have the same requirements.

```go
// NewFooCommand returns a cobra command for `foo` subcommands
func NewFooCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "foo",
		Short: "Manage foos",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version":       "1.99",
			"experimental":  "",
		},
	}
	cmd.AddCommand(
		newBarCommand(dockerCli),
		newBazCommand(dockerCli),
	)
	return cmd
}
```


Annotations exist for _orchestrators_ (kubernetes, swarm), experimental features,
and builder version. Some annotations do not require a value, in which case `nil`
should be used.

Annotation      | Example value | Description
----------------|---------------|------------------------------------------------------------------------------
`version`       | `1.40`        | Feature requires API version `1.40`, and is hidden on any older API version.
`experimental`  | `nil`         | Feature requires a daemon with experimental features enabled, and is hidden otherwise.
`no-buildkit`   | `nil`         | Feature is only used when using the legacy builder, and is hidden if BuildKit is used as builder.
`orchestrator`  | `nil`         | Feature requires an orchestrator (SwarmKit or Kubernetes) and is hidden otherwise.
`kubernetes`    | `nil`         | Feature requires the Kubernetes orchestrator, and is hidden otherwise.
`swarm`         | `nil`         | Feature requires the SwarmKit orchestrator, and is hidden otherwise.


## Validation

Docker uses a client/server architecture. As a consequence, the environment in
which the Docker CLI runs may not match the environment of the _daemon_. For
this reason, validation on the client-side should be limited, and deferred as
much as possible to the daemon.

Offloading validation to the daemon prevents validation-rules from diverging
between both, and prevents the similar code having to be maintained _twice_. Or
(worse) having validation code in the client, but unhandled by the daemon.

In addition; do not expect what's valid to never change (what's invalid today,
may be supported by the daemon, or the kernel, tomorrow).

As a rule of thumb: it's the CLI's responsibility to convert commands and arguments
into API requests. If the CLI is able to handle a value, and turn it into an a
valid API request (even if values in the request are invalid), that responsibility
is met. If the request fails, it should handle the error, and (where  needed)
present it to the user.

Some examples of validations:


OK                      | Validation                                        | Description
------------------------|---------------------------------------------------|-------------------------------------
:white_check_mark:      | Check if a required command-line argument is set  | If a command requires an argument, it is ok to check if the argument is provided.
:white_check_mark:      | Validate if a numeric value only contains numbers | This is ok. If the API requires a numeric value, the CLI should be able to convert the user-provided value.
:x:                     | Check if a name only contains allowed characters  | Rules for (e.g.) names can change over time, thus differ between daemon versions. This validation should be done by the daemon, and returned as an error.
:white_check_mark:      | Validate if a file exists before uploading it     | This is ok. The client must be able to access the file (or directory) in order to upload it.
:x:                     | Check if the host-path of a bind-mount exists     | Bind-mounts are done on the host where the daemon runs, therefore validation of these paths should not be done by the CLI.
:large_orange_diamond:  | Validate if an URL option is well-formed          | Generally this should be ok if the API expects an URL value. Consider if the validation adds value; if the CLI is able to create an API request, even if the value is invalid, validation could be offloaded to the daemon.

## Sanitizing and normalizing user input

Avoid string manipulation, unless necessary. Prefer strictness of user-provided
values over "fuzzy" matching / "guessing" user intent. Producing an error, and
loosen validation over time can be done without breaking backward compatibility
(it's guaranteed that no user was using the invalid value), whereas becoming
more strict requires a deprecation cycle ("xx is no longer valid, and support
will be removed in release XX.YY").

OK                      | Validation                                        | Description
------------------------|---------------------------------------------------|-------------------------------------
:white_check_mark:      | Trim leading and trailing whitespace              | Generally ok
:large_orange_diamond:  | Sort values                                       | Tread carefully. Sorting may be required to prevent Swarm services from being updated if no changes were made, but in some cases "order matters".
:x:                     | Strip quotes                                      | Should be handled by the shell
:x:                     | Cast strings to lowercase / uppercase             | Avoid string value manipulation if not needed.



### Commands and subcommands

#### Standardized "CRUD" commands and aliases

- `create`
- `remove`, `rm`
- `update`
- `list`, `ls`
- `inspect` (JSON)

Command                          | Aliases     | Description
---------------------------------|-------------|--------------------------------------------------
`docker <object> create`         | -           | Creates a new `<object>`
`docker <object> list`           | `ls` / `ps` | Presents a list of `<object>` (table view by default)
`docker <object> inspect <id>`   | -           | Provides low-level information about `<object> <id>` in JSON format
`docker <object> update <id>`    | -           | Updates `<object> <id>`
`docker <object> remove <id>`    | `rm`        |
`docker <object> prune`          | -           |




### Flags

#### Shorthand (single-letter) flags



#### Standardized flags

Flag              | Generally used on   | Description
------------------|---------------------|----------------------------------------------------------------
`--format` / `-f` | `list`, `inspect`   | Pretty-print objects using a Go template
`--filter` / `-f` | `list`              | Filter objects based on conditions provided
`--no-trunc`      | `list`              | List outputs can truncate columns to save screen-space. The `--no-trunc` option prints columns without truncating
`--quiet` / `-q`  | `list`              | For list outputs; only print object ID's.




#### File flags

- absolute vs relative paths
- support for stdin (CONVENTION??)

### Feedback on successful operations

Successful operations on an object should print the object's identifier on `stdout`
on success; doing so enables users to consume the output for scripting.

For example, the following command creates two volumes, using different drivers,
and attaches those volumes to a new container:

```bash
docker container run \
  --volume $(docker volume create --driver=foo):/somewhere \
  --volume $(docker volume create --driver=bar):/somewhere-else \
  busybox
```


Either `ID` or `name` are acceptable, as long as the identifier can be used
to reference the object.

Some examples of commands that follow this design:


Creating a container prints the container's `ID` on `stdout`

```bash
docker container create --name test busybox
cc1555ede50a11f02a3ef6cc8eedd9f78f6299a2f2b7efdc8a91fbefb8fc194a
```


Removing a container prints the reference that was given

```bash
docker container rm test
test

docker container rm 057d35243903e522c70ae2dfab5f706e4ea3cc1ae4c38cce955a99be3d09cfd1
057d35243903e522c70ae2dfab5f706e4ea3cc1ae4c38cce955a99be3d09cfd1
```



### Use of stdout and stderr


As a rule of thumb;

- Use `stdout` to print the _expected_ output of a command
- Where possible make `stdout` usable for scripting
- Use `stderr` non-standard output (errors), and for informational messages.

Sometimes these differences are subtle.

#### Example: usage information.

The `docker` command expects a subcommand. If no subcommand is given, the Docker
CLI prints an informational message, showing the usage information:

```bash
docker

Usage:	docker [OPTIONS] COMMAND

A self-sufficient runtime for containers
...
```

Typing `docker --help` also prints the usage information:

```bash
docker --help

Usage:	docker [OPTIONS] COMMAND

A self-sufficient runtime for containers
...
```

At a glance, both appear to be identical, but there is a difference:

In the first example, `stdout` contains the "expected" output of the `docker`
command; the `docker` command requires a subcommand, and by itself does not
produce a result.

The "usage" output is informational, and therefore printed on `stderr`, which
can be seen when discarding the `stderr` output by redirecting it to `/dev/null`;

```bash
docker 2> /dev/null
# no output (stdout contains no output)
```

When using the `--help` flag, the usage information is the _expected_ output,
and therefore printed on `stdout`. Suppressing `stderr` output shows that all
output is this time printed on `stdout`:

```bash
docker --help 2> /dev/null

Usage:	docker [OPTIONS] COMMAND

A self-sufficient runtime for containers
```


#### Practical example: docker run


logging informational messages and consuming stdout output

```bash
docker container run -d nginx:alpine | xargs docker container inspect --format 'the name of the started container is: {{.Name}}'

Unable to find image 'nginx:alpine' locally
alpine: Pulling from library/nginx
cd784148e348: Already exists
6e3058b2db8a: Already exists
7ca4d29669c1: Already exists
a14cf6997716: Already exists
Digest: sha256:385fbcf0f04621981df6c6f1abd896101eb61a439746ee2921b26abc78f45571
Status: Downloaded newer image for nginx:alpine
the name of the started container is: /vigilant_zhukovsky
```



```bash
docker container run -d nginx:alpine 2> ./err.log | xargs docker container inspect --format '{{.Name}}'
/cranky_jepsen


cat err.log
Unable to find image 'nginx:alpine' locally
alpine: Pulling from library/nginx
cd784148e348: Already exists
6e3058b2db8a: Already exists
7ca4d29669c1: Already exists
a14cf6997716: Already exists
Digest: sha256:385fbcf0f04621981df6c6f1abd896101eb61a439746ee2921b26abc78f45571
Status: Downloaded newer image for nginx:alpine
```



```
docker service create nginx:alpine
trppp1sywrkegq8e4pynjenv0
overall progress: 1 out of 1 tasks
1/1: running   [==================================================>]
verify: Service converged
```

```
docker service create nginx:alpine 2> /dev/null
clu6qi8pcg2fbnhxpuusdk0iw
overall progress: 1 out of 1 tasks
1/1: running   [==================================================>]
verify: Service converged
```

```
docker service create --detach nginx:alpine
q5d2k2bbgk6uh2rs3g4udfdh7
```




### Object identifiers

Objects can have multiple identifiers, for example, containers have both an
`ID` and a `name`, both of which must be unique (and are thus interchangeable).
Names are allowed to be mutable (for example, a container can be renamed using
the `docker container rename` command).

Objects that have both an `ID` and `name`, and where commands accept either an
`ID`, a `name` or a _partial_ `ID` (ID-prefix) should address ambiguity by
prioritizing as below:

1. Full, non-truncated `ID`
2. `name` (full match only, no prefix matching)
3. Partial `ID` (prefix matching)

If multiple objects match a given ID prefix, an error must be produced, stating
that the given prefix is ambiguous.

For example, given the following containers:

```bash
docker ps --no-trunc --format 'table {{.ID}}\t{{.Names}}'

CONTAINER ID                                                       NAMES
70d50d097b5597d8d08171e6be51cee9e15083c5b92dde134639c4105db2a40d   mycontainer01
724f22e1e32c7cc8d80383f06baebbcd0f5fec0e5592bdf6b50b2bcd8d391ab7   70d50d097b55
737ff55584e28badc5fdccd22582f6d4583d0f792cbb4924493f5d341e44a278   70d50d097b5597d8d08171e6be51cee9e15083c5b92dde134639c4105db2a40d
```

Note that:

- All containers have an `ID` starting with `7`
- The second container's name matches the first container's "short" `ID` (ID-prefix)
- The third  container's name matches the first container's full `ID`

Running `docker container inspect` using the first container's _full_ ID produces
the first container (full `ID` takes precedence over full name);

```bash
docker container inspect 70d50d097b5597d8d08171e6be51cee9e15083c5b92dde134639c4105db2a40d --format '{{.Name}}'
/mycontainer01
```

Inspecting using the second container's full name, produces that container,
_even though it also matches a prefix of the first container's ID_ (full name match
takes precedence over a partial `ID` match):

```bash
docker container inspect 70d50d097b55 --format '{{.Name}}'
/70d50d097b55
```

Inspecting using a _prefix_ produces an error, because multiple objects are matched:

```bash
docker container inspect 7 --format '{{.Name}}'
Error response from daemon: Multiple IDs found with provided prefix: 7
```

Using a longer prefix will succeed if the prefix is non-ambiguous;

```bash
docker container inspect 70 --format '{{.Name}}'
/mycontainer01
```



#### Progressbars

- automatically disabled if no terminal is detected
- also through `--quiet` / `-q` flag


#### Boolean flags should expect no value

**exception** flags that will change their default in the near future

```bash
--disable-some-feature=false
--enable-almost-deprecated-feature=false
```

#### Shorthand flag formats

- avoid microformats
- keep Windows/Linux into account
- strict > permissive (easier to be less restrictive in future than the other way round)
  - case-sensitive values (`none` != `None` != `NONE` != `nOnE`)

#### Long form (advanced) syntax

- group all options for a configuration in a single flag
- allow that flag to be set multiple times, and still being able to group those
  options together (example: `--volume-driver`, `--volume`, which prevented
  multiple drivers to be used)


### Positional arguments

Positional arguments are generally more suitable for "required" arguments, whereas flags are for "optional" arguments.

Examples:

Creating a container does not require a name to be specified (a name is generated when omitted), hence, the name
is passed as 

```bash
docker container create busybox:latest
47931878b128c40281aa0254914c3a437b2fdf91133a1620b3152e05d3c81e5d

docker container create --name mycontainer busybox:latest
908a6f802740d9a67861fc031c62d6241fbb9f16b9388114b14cf120e4f0cd68
```



## Use of stdout and stderr


- output should be useful
- 


## Exit codes

- exit code for list commands (exit code 0 for "no results")


## Filtering

## Formatting output

- Discuss `JSON` output option
- Presentation should not be in `JSON` output (or the API for that matter)
- Configurable in `~/.docker/config.json`


