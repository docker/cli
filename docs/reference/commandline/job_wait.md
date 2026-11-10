# docker job wait

<!---MARKER_GEN_START-->
Block until a run of a job reaches a terminal state, then exit with the run's exit code.

The job must exist: names of removed jobs do not resolve. Without --run, the
latest run is awaited; for a schedule job that has not run yet, that means
blocking until its next scheduled fire.

A run stopped by a timeout makes the command fail with status 124, and one
stopped by "docker job cancel" with status 130, whatever the exit code of its
container. A run whose container never started fails with status 1.

With --condition running, the ID of the run is printed once it is running. A
run that already finished is reported as usual, by its exit code, so scripts
capturing the output must check which one they got.

### Options

| Name          | Type     | Default | Description                                                                                    |
|:--------------|:---------|:--------|:-----------------------------------------------------------------------------------------------|
| `--condition` | `string` |         | Condition to wait for: `terminal` or `running`, i.e. the run left pending (default `terminal`) |
| `--run`       | `string` |         | Run to wait for: a run ID, or `latest` (default `latest`)                                      |


<!---MARKER_GEN_END-->

