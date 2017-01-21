/*
Package fsstat provides a MetricSet for fetching aggregated filesystem stats.

An example event looks as following:

{
  "@timestamp": "2016-05-25T19:47:57.216Z",
  "beat": {
    "hostname": "host.example.com",
    "name": "host.example.com"
  },
  "metricset": {
    "module": "system",
    "name": "fsstat",
    "rtt": 119
  },
  "system": {
    "fsstat": {
      "count": 4,
      "total_files": 60982408,
      "total_size": {
        "free": 14127017984,
        "total": 249779535360,
        "used": 235652517376
      }
    }
  },
  "type": "metricsets"
}

*/
package fsstat
