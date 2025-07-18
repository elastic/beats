---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-vsphere-host.html
---

% This file is generated! See scripts/docs_collector.py

# vSphere host metricset [metricbeat-metricset-vsphere-host]

This is the `host` metricset of the vSphere module.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-vsphere.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2022-09-06T06:41:22.128Z",
    "metricset": {
        "name": "host",
        "period": 10000
    },
    "service": {
        "address": "https://localhost:8989/sdk",
        "type": "vsphere"
    },
    "event": {
        "module": "vsphere",
        "duration": 23519250,
        "dataset": "vsphere.host"
    },
    "vsphere": {
        "host": {
            "triggered_alarms": [
                {
                    "status": "red",
                    "triggered_time": "2024-09-09T13:23:00.786Z",
                    "description": "Default alarm to monitor system boards.  See the host's Hardware Status tab for more details.",
                    "entity_name": "121.0.0.0",
                    "name": "Host hardware system board status",
                    "id": "alarm-121.host-12"
                },
                {
                    "triggered_time": "2024-09-09T13:23:00.786Z",
                    "description": "Default alarm to monitor storage.  See the host's Hardware Status tab for more details.",
                    "entity_name": "121.0.0.0",
                    "name": "Host storage status",
                    "id": "alarm-124.host-12",
                    "status": "red"
                },
                {
                    "entity_name": "121.0.0.0",
                    "name": "Host memory usage",
                    "id": "alarm-4.host-12",
                    "status": "yellow",
                    "triggered_time": "2024-08-28T10:31:26.621Z",
                    "description": "Default alarm to monitor host memory usage"
                },
                {
                    "name": "CPU Utilization",
                    "id": "alarm-703.host-12",
                    "status": "red",
                    "triggered_time": "2024-08-28T10:31:26.621Z",
                    "description": "",
                    "entity_name": "121.0.0.0"
                }
            ],
            "cpu": {
                "used": {
                    "mhz": 67
                },
                "total": {
                    "mhz": 4588
                },
                "free": {
                    "mhz": 4521
                }
            },
            "disk": {
                "capacity": {
                    "usage": {
                        "bytes": 0
                    }
                },
                "devicelatency": {
                    "average": {
                        "ms": 0
                    }
                },
                "latency": {
                    "total": {
                        "ms": 18
                    }
                },
                "total": {
                    "bytes": 262000
                },
                "read": {
                    "bytes": 13000
                },
                "write": {
                    "bytes": 248000
                }
            },
            "memory": {
                "free": {
                    "bytes": 2822230016
                },
                "total": {
                    "bytes": 4294430720
                },
                "used": {
                    "bytes": 1472200704
                }
            },
            "network": {
                "bandwidth": {
                    "total": {
                        "bytes": 372000
                    },
                    "transmitted": {
                        "bytes": 0
                    },
                    "received": {
                        "bytes": 371000
                    }
                },
                "packets": {
                    "received": {
                        "count": 9463
                    },
                    "errors": {
                        "transmitted": {
                            "count": 0
                        },
                        "received": {
                            "count": 0
                        },
                        "total": {
                            "count": 0
                        }
                    },
                    "multicast": {
                        "total": {
                            "count": 6679
                        },
                        "transmitted": {
                            "count": 0
                        },
                        "received": {
                            "count": 6679
                        }
                    },
                    "dropped": {
                        "received": {
                            "count": 0
                        },
                        "total": {
                            "count": 0
                        },
                        "transmitted": {
                            "count": 0
                        }
                    },
                    "transmitted": {
                        "count": 54
                    }
                }
            },
            "vm": {
                "count": 2,
                "names": [
                    "DC0_H0_VM0",
                    "DC0_H0_VM1"
                ]
            },
            "datastore": {
                "count": 1,
                "names": [
                    "LocalDS_0"
                ]
            },
            "network_names": [
                "VM Network"
            ],
            "id": "host-0",
            "name": "DC0_H0",
            "status": "green",
            "uptime": 1728865
        }
    }
}
```
