---
navigation_title: "Monitors"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/configuration-heartbeat-options.html
---

# Configure Heartbeat monitors [configuration-heartbeat-options]


To configure Heartbeat define a set of `monitors` to check your remote hosts. Specify monitors either directly inside the `heartbeat.yml` config file, or in external dynamically loaded files located in the directory referenced by `heartbeat.config.monitors.path`. One advantage of using external files is that these can be automatically reloaded without stopping the Heartbeat process.

Each `monitor` item is an entry in a yaml list, and so begins with a dash (-). You can define the type of monitor to use, the hosts to check, and other optional settings that control Heartbeat behavior.

The following example configures three monitors checking via the `icmp`, `tcp`, and `http` protocols directly inside the `heartbeat.yml` file, and demonstrates how to use TCP Echo and HTTP response verification:

```yaml
# heartbeat.yml
heartbeat.monitors:
- type: icmp
  id: ping-myhost
  name: My Host Ping
  hosts: ["myhost"]
  schedule: '*/5 * * * * * *'
- type: tcp
  id: myhost-tcp-echo
  name: My Host TCP Echo
  hosts: ["myhost:777"]  # default TCP Echo Protocol
  check.send: "Check"
  check.receive: "Check"
  schedule: '@every 5s'
- type: http
  id: service-status
  name: Service Status
  service.name: my-apm-service-name
  hosts: ["http://localhost:80/service/status"]
  check.response.status: [200]
  schedule: '@every 5s'
heartbeat.scheduler:
  limit: 10
```

Using the `heartbeat.yml` configuration file is convenient, but has two drawbacks: it can become hard to manage with large numbers of monitors, and it will not reload heartbeat automatically when its contents changes.

Define monitors via the `heartbeat.config.monitors` to prevent those issues from happening to you. To do so you would instead have your `heartbeat.yml` file contain the following:

```yaml
# heartbeat.yml
heartbeat.config.monitors:
  # Directory + glob pattern to search for configuration files
  path: /path/to/my/monitors.d/*.yml
  # If enabled, heartbeat will periodically check the config.monitors path for changes
  reload.enabled: true
  # How often to check for changes
  reload.period: 1s
```

Then, define one or more files in the directory pointed to by `heartbeat.config.monitors.path`. You may specify multiple monitors in a given file if you like. The contents of these files is monitor definitions only, e.g. what is normally under the `heartbeat.monitors` section of `heartbeat.yml`. See below for an example

```yaml
# /path/to/my/monitors.d/localhost_service_check.yml
- type: http
  id: service-status
  name: Service Status
  hosts: ["http://localhost:80/service/status"]
  check.response.status: [200]
  schedule: '@every 5s'
```


## Monitor types [monitor-types]

You can configure Heartbeat to use the following monitor types:

**[`icmp`](/reference/heartbeat/monitor-icmp-options.md)**
:   Uses an ICMP (v4 and v6) Echo Request to ping the configured hosts. Requires special permissions or root access.

**[`tcp`](/reference/heartbeat/monitor-tcp-options.md)**
:   Connects via TCP and optionally verifies the endpoint by sending and/or receiving a custom payload.

**[`http`](/reference/heartbeat/monitor-http-options.md)**
:   Connects via HTTP and optionally verifies that the host returns the expected response. Will use `Elastic-Heartbeat` as the user agent product.

The `tcp` and `http` monitor types both support SSL/TLS and some proxy settings.

::::{note}
**Looking for browser monitor options?**  {{heartbeat}} browser checks are in beta and will never be made generally available.

If you want to set up browser checks, we highly recommend using Synthetics via [Projects](docs-content://solutions/observability/synthetics/create-monitors-with-projects.md) or the [Synthetics app in {{kib}}](docs-content://solutions/observability/synthetics/create-monitors-ui.md) to create and manage browser monitors.

If you need to refer to how beta {{heartbeat}} browser checks were set up previously, refer to the [Browser options documentation for 8.7](https://www.elastic.co/guide/en/beats/heartbeat/8.7/monitor-browser-options.html).
::::






