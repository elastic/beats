/*
Package filesystem provides a MetricSet implementation that fetches metrics
for each of the mounted file systems.

An example event looks as following:

{
  "@timestamp": "2016-05-25T20:51:36.813Z",
  "beat": {
    "hostname": "host.example.com",
    "name": "host.example.com"
  },
  "metricset": {
    "module": "system",
    "name": "filesystem",
    "rtt": 55
  },
  "system": {
    "filesystem": {
      "avail": 13838553088,
      "device_name": "/dev/disk1",
      "files": 60981246,
      "free": 14100697088,
      "free_files": 3378553,
      "mount_point": "/",
      "total": 249779191808,
      "used": 235678494720,
      "used_p": 0.9435
    }
  },
  "type": "metricsets"
}

*/
package filesystem
