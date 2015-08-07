# topbeat

Topbeat is a [Beat](https://www.elastic.co/products/beats) that periodically
reads system wide and per process CPU and memory statistics and indexes them in
Elasticsearch.

This is quite early stage and not yet released.

There are two types of documents exported:
- type: system for system wide statistics
- type: proc for per process statistics. One per process is generated.
 
Example documents:

    {
        count":1,
        "proc.cpu":{
          "user":20,
          "percent":0.983284169124877,
          "system":0,
          "total":20,
          "start_time":"20:20"
        },
        "proc.mem":{
          "size":333772,
          "rss":6316,
          "share":2996
        },
        "proc.name":"topbeat",
        "proc.pid":13954,
        "proc.ppid":10027,
        "proc.state":"sleeping",
        "shipper":"vagrant-ubuntu-trusty-64",
        "timestamp":"2015-08-06T20:20:34.089Z",
        "type":"proc"
    }


    {
        "count":1,
        "cpu":{
          "user":33030,
          "nice":0,
          "system":16392,
          "idle":12134029,
          "iowait":154,
          "irq":10,
          "softirq":2038,
          "steal":0
        },
        "load":{
          "load1":0.85,
          "load5":0.42,
          "load15":0.21
        },
        "mem":{
          "total":501748,
          "used":339176,
          "free":162572,
          "actual_used":135904,
          "actual_free":365844
        },
        "shipper":"vagrant-ubuntu-trusty-64",
        "swap":{
          "total":0,
          "used":0,
          "free":0,
          "actual_used":0,
          "actual_free":0
        },
        "timestamp":"2015-08-06T20:20:33.062Z",
        "type":"system"
    }
