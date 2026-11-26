# docker job

<!---MARKER_GEN_START-->
Manage jobs, one-shot or scheduled containers run by the daemon.

This requires the daemon's jobs extension, which is disabled by default: start
dockerd with --extension-enable org.mobyproject.jobs.v0.

### Subcommands

| Name                        | Description                                      |
|:----------------------------|:-------------------------------------------------|
| [`cancel`](job_cancel.md)   | Cancel a job's in-flight run                     |
| [`inspect`](job_inspect.md) | Display detailed information on a job            |
| [`logs`](job_logs.md)       | Fetch the logs of a job's most recent run        |
| [`ls`](job_ls.md)           | List jobs                                        |
| [`pause`](job_pause.md)     | Suppress a job's schedule trigger                |
| [`prune`](job_prune.md)     | Remove idle manual jobs                          |
| [`rm`](job_rm.md)           | Remove a job                                     |
| [`run`](job_run.md)         | Run a job                                        |
| [`runs`](job_runs.md)       | List a job's runs                                |
| [`unpause`](job_unpause.md) | Re-arm a paused job's schedule trigger           |
| [`wait`](job_wait.md)       | Block until a job's run reaches a terminal state |



<!---MARKER_GEN_END-->

