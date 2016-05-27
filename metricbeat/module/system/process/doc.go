/*
Package process collects metrics about the running processes using information
from the operating system.

An example event looks as following:
{
  "@timestamp": "2016-05-25T20:57:51.854Z",
  "beat": {
    "hostname": "host.example.com",
    "name": "host.example.com"
  },
  "metricset": {
    "module": "system",
    "name": "process",
    "rtt": 12269
  },
  "system": {
    "process": {
      "cmdline": "/System/Library/CoreServices/ReportCrash",
      "cpu": {
        "start_time": "22:57",
        "total_p": 0
      },
      "mem": {
        "rss": 27123712,
        "rss_pct": 0.0016,
        "share": 0,
        "size": 2577522688
      },
      "name": "ReportCrash",
      "pid": 97801,
      "ppid": 1,
      "state": "running",
      "username": "elastic"
    }
  },
  "type": "metricsets"
}
*/
package process
