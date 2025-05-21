The Docker `healthcheck` metricset collects healthcheck status metrics about running Docker containers.

Healthcheck data will only be available from docker containers where the docker `HEALTHCHECK` instruction has been used to build the docker image.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
