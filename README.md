[![Jenkins Build Status](http://build-eu-00.elastic.co/job/topbeat/badge/icon)](http://build-eu-00.elastic.co/job/topbeat/)
[![Build Status](https://travis-ci.org/elastic/topbeat.svg?branch=master)](https://travis-ci.org/elastic/topbeat)
[![codecov.io](http://codecov.io/github/elastic/topbeat/coverage.svg?branch=master)](http://codecov.io/github/elastic/topbeat?branch=master)

# topbeat

Topbeat is the [Beat](https://www.elastic.co/products/beats) used for
server monitoring. It is a lightweight agent that installed on your servers,
reads periodically system wide and per process CPU and memory statistics and indexes them in
Elasticsearch.

## Exported fields

There are two types of documents exported:
- `type: system` for system wide statistics
- `type: proc` for per process statistics. One per process is generated.

<pre>

{
  "_index": "topbeat-2015.10.30",
  "_type": "proc",
  "_id": "AVC5EgksdTLnTMyb6dAs",
  "_score": null,
  "_source": {
    "@timestamp": "2015-10-30T14:06:17.412Z",
    "count": 1,
    "proc": {
      "cpu": {
        "user": 1998400,
        "user_p": 0.01,
        "system": 603688,
        "total": 2602088,
        "start_time": "Oct28"
      },
      "mem": {
        "size": 1106194432,
        "rss": 214093824,
        "rss_p": 0.01,
        "share": 0
      },
      "name": "zoom.us",
      "pid": 234,
      "ppid": 1,
      "state": "running"
    },
    "shipper": "mar.localdomain",
    "type": "proc"
  },
  "fields": {
    "@timestamp": [
      1446213977412
    ]
  },
  "highlight": {
    "type": [
      "@kibana-highlighted-field@proc@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1446213977412
  ]
}

{
  "_index": "topbeat-2015.10.30",
  "_type": "system",
  "_id": "AVC5EfWidTLnTMyb6c7B",
  "_score": null,
  "_source": {
    "@timestamp": "2015-10-30T14:06:12.405Z",
    "count": 1,
    "cpu": {
      "user": 1737666,
      "user_p": 0.05,
      "nice": 0,
      "system": 1483741,
      "system_p": 0.05,
      "idle": 19489227,
      "iowait": 0,
      "irq": 0,
      "softirq": 0,
      "steal": 0
    },
    "load": {
      "load1": 2.26513671875,
      "load5": 2.1513671875,
      "load15": 1.96728515625
    },
    "mem": {
      "total": 17179869184,
      "used": 11818733568,
      "free": 5361135616,
      "used_p": 0.69,
      "actual_used": 9940553728,
      "actual_free": 7239315456
    },
    "shipper": "mar.localdomain",
    "swap": {
      "total": 1073741824,
      "used": 219414528,
      "free": 854327296,
      "used_p": 0.2,
      "actual_used": 0,
      "actual_free": 0
    },
    "type": "system"
  },
  "fields": {
    "@timestamp": [
      1446213972405
    ]
  },
  "highlight": {
    "type": [
      "@kibana-highlighted-field@system@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1446213972405
  ]
}

{
  "_index": "topbeat-2015.10.30",
  "_type": "filesystem",
  "_id": "AVC5EgksdTLnTMyb6dA2",
  "_score": null,
  "_source": {
    "@timestamp": "2015-10-30T14:06:17.412Z",
    "count": 1,
    "fs": {
      "device_name": "devfs",
      "total": 347648,
      "used": 347648,
      "used_p": 1,
      "free": 0,
      "avail": 0,
      "files": 1176,
      "free_files": 0,
      "mount_point": "/dev"
    },
    "shipper": "mar.localdomain",
    "type": "filesystem"
  },
  "fields": {
    "@timestamp": [
      1446213977412
    ]
  },
  "highlight": {
    "type": [
      "@kibana-highlighted-field@filesystem@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1446213977412
  ]
}
</pre>

## Elasticsearch template

To apply topbeat template:

    curl -XPUT 'http://localhost:9200/_template/topbeat' -d@etc/topbeat.template.json
