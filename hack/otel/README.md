# Sample stack for testing OTEL functionality with the CLI

To test the OTEL functionality present in the CLI, you can spin up a small demo compose stack that includes:
- an OTEL collector container;
- a Prometheus container;
- an Aspire Dashboard container

The `hack/otel` directory contains the compose file with the services configured, along with 2 basic configuration files: one for the OTEL collector and one for Prometheus.

## How can I use it?

1) Start the compose stack by running `docker compose up -d` in the `hack/otel/` directory;
2) Export the env var used to override the OTLP endpoint:  
  `export DOCKER_CLI_OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` (if running the CLI in a devcontainer or in other ways, you might have to change how you pass this env var);
3) Run the CLI to send some metrics to the endpoint;
4) Browse Prometheus at `http://localhost:9091/graph` or the Aspire Dashboard at  `http://localhost:18888/metrics`;
5) In Prometheus, query `command_time_milliseconds_total` to see some metrics. In Aspire, select the resource in the dropdown.

> **Note**: The precise steps may vary based on how you're working on the codebase (buiding a binary and executing natively, running/debugging in a devcontainer, running the normal CLI as usual, etc... )

## Cleanup?

Run `docker compose down` in the `hack/otel/` directory.

You can also run `unset DOCKER_CLI_OTEL_EXPORTER_OTLP_ENDPOINT` to get rid of the OTLP override from your environment.
