/*
Package filesystem provides a MetricSet implementation that fetches metrics
for each of the mounted file systems.

An example event looks as following:

{
    "@timestamp": "2016-05-23T08:05:34.853Z",
    "beat": {
        "hostname": "host.example.com",
        "name": "host.example.com"
    },
    "@metadata": {
        "beat": "noindex",
        "type": "doc"
    },
    "metricset": {
        "module": "system",
        "name": "filesystem",
        "rtt": 115
    },
    "system": {
        "filesystem": {
            "available": 105569656832,
            "device_name": "/dev/disk1",
            "type": "hfs",
            "files": 4294967279
            "free": 105831800832,
            "free_files": 4292793781,
            "mount_point": "/",
            "total": 249779191808,
            "used": {
                "bytes": 143947390976,
                "pct": 0.5763
            },
        }
    }
}

*/
package filesystem
