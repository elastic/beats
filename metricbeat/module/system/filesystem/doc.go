/*
Package filesystem provides a MetricSet implementation that fetches metrics
for each of the mounted file systems.

An example event looks as following:

    {
      "@timestamp": "2016-04-26T19:30:19.475Z",
      "beat": {
        "hostname": "ruflin",
        "name": "ruflin"
      },
      "metricset": "filesystem",
      "module": "system",
      "rtt": 434,
      "system-filesystem": {
        "avail": 41159540736,
        "device_name": "/dev/disk1",
        "files": 60981246,
        "free": 41421684736,
        "free_files": 10048716,
        "mount_point": "/",
        "total": 249779191808,
        "used": 208357507072,
        "used_p": 0.83
      },
      "type": "metricsets"
    }
*/
package filesystem
