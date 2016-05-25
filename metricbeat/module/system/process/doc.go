/*
Package process collects metrics about the running processes using information
from the operating system.

An example event looks as following:

    {
      "@timestamp": "2016-04-26T19:24:19.108Z",
      "beat": {
        "hostname": "host.example.com",
        "name": "host.example.com"
      },
      "metricset": "process",
      "module": "system",
      "rtt": 20982,
      "system-process": {
        "cmdline": ".\/metricbeat -e -d * -c metricbeat.dev.yml",
        "cpu": {
          "start_time": "21:24",
          "system": 32,
          "total": 79,
          "total_p": 1.2791,
          "user": 47
        },
        "mem": {
          "rss": 11538432,
          "rss_p": 0,
          "share": 0,
          "size": 587196518400
        },
        "name": "metricbeat",
        "pid": 27769,
        "ppid": 26608,
        "state": "running",
        "username": "ruflin"
      },
      "type": "metricsets"
    }
*/
package process
