# topbeat

Topbeat is a [Beat](https://www.elastic.co/products/beats) that periodically
reads system wide and per process CPU and memory statistics and indexes them in
Elasticsearch.

This is quite early stage and not yet released.

## Example document

        {
          "count": 1,
          "cpu": {
            "user": 1,
            "nice": 0,
            "system": 0,
            "idle": 99,
            "iowait": 0,
            "irq": 0,
            "softirq": 0,
            "steal": 0,
            "guest": 0,
            "guestnice": 0
          },
          "load": {
            "load1": 0.06,
            "load5": 0.04,
            "load15": 0.05
          },
          "mem": {
            "total": 501748,
            "available": 375584,
            "used": 301740,
            "used_p": 0,
            "free": 200008,
            "buffers": 21296,
            "cached": 154280,
            "active": 175688,
            "inactive": 86024
          },
          "procs": [
            {
              "pid": 1,
              "ppid": 0,
              "name": "init",
              "state": "sleeping",
              "utime": 0.2,
              "stime": 0.57,
              "vsize": 33600,
              "rss": 2936,
              "num_threads": 1
            },
            {
              "pid": 10,
              "ppid": 2,
              "name": "rcuob/0",
              "state": "sleeping",
              "utime": 0,
              "stime": 0,
              "vsize": 0,
              "rss": 0,
              "num_threads": 1
            },

            ...
          ]
        }
