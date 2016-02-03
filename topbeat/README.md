# Topbeat

Topbeat is the [Beat](https://www.elastic.co/products/beats) used for
server monitoring. It is a lightweight agent that installed on your servers,
reads periodically system wide and per process CPU and memory statistics and indexes them in
Elasticsearch.

## Documentation

You can find the documentation on the [elastic.co](https://www.elastic.co/guide/en/beats/topbeat/current/index.html) website.

## Exported fields

Topbeat exports the following types of JSON documents based on the statistics
that it's configured to capture:

- `type: system` for system-wide statistics
- `type: process` for per-process statistics
- `type: filesystem` for disk usage statistics

By default, Topbeat captures all three types of statistics. As you can see in
the following examples, the content of the JSON document varies depending on the
type.

### System-Wide Statistics

Topbeat exports one JSON document for the system. For example:

<pre>
{
  "@timestamp": "2016-02-01T20:16:49.480Z",
  "beat": {
    "hostname": "MacBook-Pro.local",
    "name": "MacBook-Pro.local"
  },
  "count": 1,
  "cpu": {
    "user": 2985331,
    "user_p": 0,
    "nice": 0,
    "system": 1727403,
    "system_p": 0,
    "idle": 25915908,
    "iowait": 0,
    "irq": 0,
    "softirq": 0,
    "steal": 0
  },
  "cpu0": {
    "user": 2985331,
    "user_p": 0,
    "nice": 0,
    "system": 1727403,
    "system_p": 0,
    "idle": 25915908,
    "iowait": 0,
    "irq": 0,
    "softirq": 0,
    "steal": 0
  },
  "load": {
    "load1": 1.52392578125,
    "load5": 1.79736328125,
    "load15": 1.98291015625
  },
  "mem": {
    "total": 17179869184,
    "used": 8868311040,
    "free": 8311558144,
    "used_p": 0.52,
    "actual_used": 8355057664,
    "actual_free": 8824811520,
    "actual_used_p": 0.49
  },
  "swap": {
    "total": 2147483648,
    "used": 736624640,
    "free": 1410859008,
    "used_p": 0.34
  },
  "type": "system"
}
</pre>

If you set `cpu_per_core: true`, the output also includes CPU usage per core
statistics grouped under `cpus`.

### Per-Process Statistics

Topbeat exports one document per process. For example:

<pre>
{
  "@timestamp": "2016-02-01T20:16:49.499Z",
  "beat": {
    "hostname": "MacBook-Pro.local",
    "name": "MacBook-Pro.local"
  },
  "count": 1,
  "proc": {
    "cpu": {
      "user": 1,
      "total_p": 0,
      "system": 1,
      "total": 2,
      "start_time": "15:59"
    },
    "mem": {
      "size": 2491260928,
      "rss": 774144,
      "rss_p": 0,
      "share": 0
    },
    "name": "less",
    "pid": 20366,
    "ppid": 10392,
    "state": "running"
  },
  "type": "process"
}
</pre>

### Disk Usage Statistics

Topbeat exports one document per mount point. For example:

<pre>
{
  "@timestamp": "2016-02-01T20:16:49.499Z",
  "beat": {
    "hostname": "MacBook-Pro.local",
    "name": "MacBook-Pro.local"
  },
  "count": 1,
  "fs": {
    "device_name": "devfs",
    "total": 198656,
    "used": 198656,
    "used_p": 1,
    "free": 0,
    "avail": 0,
    "files": 677,
    "free_files": 0,
    "mount_point": "/dev"
  },
  "type": "filesystem"
}
</pre>

## Elasticsearch template

To apply the Topbeat template:

    curl -XPUT 'http://localhost:9200/_template/topbeat' -d@etc/topbeat.template.json
