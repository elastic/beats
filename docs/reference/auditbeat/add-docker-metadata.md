---
navigation_title: "add_docker_metadata"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/add-docker-metadata.html
applies_to:
  stack: ga
  serverless: ga
---

# Add Docker metadata [add-docker-metadata]


The `add_docker_metadata` processor annotates each event with relevant metadata from Docker containers. At startup it detects a docker environment and caches the metadata. The events are annotated with Docker metadata, only if a valid configuration is detected and the processor is able to reach Docker API.

{applies_to}`stack: ga 9.5+` If Docker is unavailable at startup, the processor retries the connection every `wait_for_metadata_retry_period` (defaults to `10s`) until `wait_for_metadata_timeout` expires. By default, retries stop after `30s`. Set `wait_for_metadata` to `true` to block startup until Docker metadata is available. Set `wait_for_metadata_timeout` to `0` to retry indefinitely.

Each event is annotated with:

* Container ID
* Name
* Image
* Labels

::::{note}
When running Auditbeat in a container, you need to provide access to Docker’s unix socket in order for the `add_docker_metadata` processor to work. You can do this by mounting the socket inside the container. For example:

`docker run -v /var/run/docker.sock:/var/run/docker.sock ...`

To avoid privilege issues, you may also need to add `--user=root` to the `docker run` flags. Because the user must be part of the docker group in order to access `/var/run/docker.sock`, root access is required if Auditbeat is running as non-root inside the container.

If Docker daemon is restarted the mounted socket will become invalid and metadata will stop working, in these situations there are two options:

* Restart Auditbeat every time Docker is restarted
* Mount the entire `/var/run` directory (instead of just the socket)

::::


```yaml
processors:
  - add_docker_metadata:
      host: "unix:///var/run/docker.sock"
      #match_fields: ["system.process.cgroup.id"]
      #match_pids: ["process.pid", "process.parent.pid"]
      #match_source: true
      #match_source_index: 4
      #match_short_id: true
      #cleanup_timeout: 60
      #labels.dedot: true
      #wait_for_metadata: false
      #wait_for_metadata_timeout: 30s
      #wait_for_metadata_retry_period: 10s
      # To connect to Docker over TLS you must specify a client and CA certificate.
      #ssl:
      #  certificate_authority: "/etc/pki/root/ca.pem"
      #  certificate:           "/etc/pki/client/cert.pem"
      #  key:                   "/etc/pki/client/cert.key"
```

It has the following settings:

`host`
:   (Optional) Docker socket (UNIX or TCP socket). It uses `unix:///var/run/docker.sock` by default.

`ssl`
:   (Optional) SSL configuration to use when connecting to the Docker socket.

`match_fields`
:   (Optional) A list of fields to match a container ID, at least one of them should hold a container ID to get the event enriched.

`match_pids`
:   (Optional) A list of fields that contain process IDs. If the process is running in Docker then the event will be enriched. The default value is `["process.pid", "process.parent.pid"]`.

`match_source`
:   (Optional) Match container ID from a log path present in the `log.file.path` field. Enabled by default.

`match_short_id`
:   (Optional) Match container short ID from a log path present in the `log.file.path` field. Disabled by default. This allows to match directories names that have the first 12 characters of the container ID. For example, `/var/log/containers/b7e3460e2b21/*.log`.

`match_source_index`
:   (Optional) Index in the source path split by `/` to look for container ID. It defaults to 4 to match `/var/lib/docker/containers/<container_id>/*.log`

`cleanup_timeout`
:   (Optional) Time of inactivity to consider we can clean and forget metadata for a container, 60s by default.

`hostfs`
:   (Optional) Specifies the mount point of the host’s filesystem, which can be used to monitor a host from within a container.

`labels.dedot`
:   (Optional) If set to `true`, replaces dots in labels with `_`. Defaults to `true`.

`wait_for_metadata` {applies_to}`stack: ga 9.5+`
:   (Optional) When `true`, startup is blocked while the processor retries connecting to Docker until metadata is available. If the processor can't connect to Docker within the duration set in `wait_for_metadata_timeout`, startup fails and the process exits. When `false`, the processor starts immediately and retries the connection asynchronously until the timeout expires. Defaults to `false`.

`wait_for_metadata_timeout` {applies_to}`stack: ga 9.5+`
:   (Optional) The maximum time allowed for the initial Docker connection attempt and subsequent retries at startup. Applies regardless of `wait_for_metadata`. To retry the connection indefinitely, set to `0`. Defaults to `30s`.

`wait_for_metadata_retry_period` {applies_to}`stack: ga 9.5+`
:   (Optional) How long to wait between Docker connection retry attempts after a failed attempt. Defaults to `10s`.
