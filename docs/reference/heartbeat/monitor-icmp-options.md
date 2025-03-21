---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/monitor-icmp-options.html
---

# ICMP options [monitor-icmp-options]

Also see [Common monitor options](/reference/heartbeat/monitor-options.md).

The options described here configure Heartbeat to use ICMP (v4 and v6) Echo Requests to check the configured hosts. Please note that on most platforms you must execute Heartbeat with elevated permissions to perform ICMP pings.

On Linux, regular users may perform pings if the right file capabilities are set. Run `sudo setcap cap_net_raw+eip /path/to/heartbeat` to  grant Heartbeat ping capabilities on Linux.

The binary has the correct capabilities already set on the container image, however your container runtime must allow the use of these privileges for them to be used. On docker this can be achieved with `--cap-add=NET_RAW`.

Other platforms may require Heartbeat to run as root or administrator to execute pings.

Example configuration:

```yaml
- type: icmp
  id: ping-myhost
  name: My Host Ping
  hosts: ["myhost"]
  schedule: '*/5 * * * * * *'
```


## `hosts` [monitor-icmp-hosts]

A list of hosts to ping.


## `wait` [monitor-icmp-wait]

The duration to wait before emitting another ICMP Echo Request if no response is received. The default is 1 second (1s).

