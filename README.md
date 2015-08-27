# topbeat

Topbeat is the [Beat](https://www.elastic.co/products/beats) used for
server monitoring. It is a lightweight agent that installed on your servers,
reads periodically system wide and per process CPU and memory statistics and indexes them in
Elasticsearch.

This is quite early stage and not yet released.

## Exported fields

There are two types of documents exported:
- `type: system` for system wide statistics
- `type: proc` for per process statistics. One per process is generated.

    {
      "_index": "topbeat-2015.08.27",
      "_type": "system",
      "_id": "AU9uLFz1sHueqcsbvZmh",
      "_source": {
        "count": 1,
        "cpu": {
          "user": 9129745,
          "user_p": 85.11,
          "nice": 0,
          "system": 1878287,
          "system_p": 14.64,
          "idle": 17645082,
          "iowait": 0,
          "irq": 0,
          "softirq": 0,
          "steal": 0
        },
        "load": {
          "load1": 6.52392578125,
          "load5": 6.5341796875,
          "load15": 7.10791015625
        },
        "mem": {
          "total": 16777216,
          "used": 11466136,
          "free": 5311080,
          "used_p": 68.34,
          "actual_used": 11064836,
          "actual_free": 5712380
        },
        "shipper": "mar.localdomain",
        "swap": {
          "total": 2147483648,
          "used": 931135488,
          "free": 1216348160,
          "used_p": 43.36,
          "actual_used": 0,
          "actual_free": 0
        },
        "timestamp": "2015-08-27T08:00:45.044Z",
        "type": "system"
      }
    }


    {
      "_index": "topbeat-2015.08.27",
      "_type": "proc",
      "_id": "AU9uLi3MsHueqcsbve66",
      "_source": {
        "count": 1,
        "proc.cpu": {
          "user": 21149958,
          "user_p": 334.62,
          "system": 272950,
          "total": 21422908,
          "start_time": "22:29"
        },
        "proc.mem": {
          "size": 145164088,
          "rss": 276,
          "rss_p": 0,
          "share": 0
        },
        "proc.name": "burn",
        "proc.pid": 24090,
        "proc.ppid": 24087,
        "proc.state": "running",
        "shipper": "mar.localdomain",
        "timestamp": "2015-08-27T08:02:44.077Z",
        "type": "proc"
      }
    }

## Elasticsearch template

To apply topbeat template:

    curl -XPUT 'http://localhost:9200/_template/topbeat' -d@etc/topbeat.template.json
