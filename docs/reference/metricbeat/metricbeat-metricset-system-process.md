---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-system-process.html
---

# System process metricset [metricbeat-metricset-system-process]

The System `process` metricset provides process statistics. One document is provided for each process.

This metricset is available on:

* FreeBSD
* Linux
* macOS
* Windows


## Configuration [_configuration_12]

**`processes`**
:   When the `process` metricset is enabled, you can use the `processes` option to define a list of regexp expressions to filter the processes that are reported. For more complex filtering, you should use the `processors` configuration option. See [Processors](/reference/metricbeat/filtering-enhancing-data.md) for more information.

    The following example config returns metrics for all processes:

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: ["process"]
      processes: ['.*']
    ```


**`process.cgroups.enabled`**
:   When the `process` metricset is enabled, you can use this boolean configuration option to disable cgroup metrics. By default cgroup metrics collection is enabled.

    The following example config disables cgroup metrics on Linux.

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: ["process"]
      process.cgroups.enabled: false
    ```


**`process.cmdline.cache.enabled`**
:   This metricset caches the command line args for a running process by default. This means if you alter the command line for a process while this metricset is running, these changes are not detected. Caching can be disabled by setting `process.cmdline.cache.enabled: false` in the configuration.

**`process.env.whitelist`**
:   This metricset can collect the environment variables that were used to start the process. This feature is available on Linux, Darwin, and FreeBSD. No environment variables are collected by default because they could contain sensitive information. You must configure the environment variables that you wish to collect by specifying a list of regular expressions that match the variable name.

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: ["process"]
      process.env.whitelist:
      - '^PATH$'
      - '^SSH_.*'
    ```


**`process.include_cpu_ticks`**
:   By default the cumulative CPU tick values are not reported by this metricset (only percentages are reported). Setting this option to true will enable the reporting of the raw CPU tick values (for user, system, and total CPU time).

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: ["process"]
      process.include_cpu_ticks: true
    ```


**`process.include_per_cpu`**
:   By default metrics per cpu are reported when available. Setting this option to false will disable the reporting of these metrics.

**`process.include_top_n`**
:   These options allow you to filter out all processes that are not in the top N by CPU or memory, in order to reduce the number of documents created. If both the `by_cpu` and `by_memory` options are used, the union of the two sets is included.

**`process.include_top_n.enabled`**
:   Set to false to disable the top N feature and include all processes, regardless of the other options. The default is `true`, but nothing is filtered unless one of the other options (`by_cpu` or `by_memory`) is set to a non-zero value.

**`process.include_top_n.by_cpu`**
:   How many processes to include from the top by CPU. The processes are sorted by the `system.process.cpu.total.pct` field. The default is 0.

**`process.include_top_n.by_memory`**
:   How many processes to include from the top by memory. The processes are sorted by the `system.process.memory.rss.bytes` field. The default is 0.


## Monitoring Hybrid Hierarchy Cgroups [_monitoring_hybrid_hierarchy_cgroups]

The process metricset supports both V1 and V2 (sometimes called unfied) cgroups controllers. However, on systems that are running a hybrid hierarchy, with both V1 and V2 controllers, metricbeat will only report one of the hierarchies for a given process. Is a process has both V1 and V2 hierarchies associated with it, metricbeat will check to see if the process is attached to any V2 controllers. If it is, it will report cgroups V2 metrics. If not, it will report V1 metrics.

A workaround is also required if metricbeat is running inside docker on a hybrid system. Within docker, metricbeat won’t be able to see any V2 cgroups components. If you wish to monitor cgroups V2 from within docker on a hybrid system, you must mount the unified sysfs hierarchy (usually `/sys/fs/cgroups/unified`) inside the container, and then use `system.hostfs` to specify the filesystem root within the container.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.

## Fields [_fields_242]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-system.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "event": {
        "dataset": "system.process",
        "duration": 115000,
        "module": "system"
    },
    "metricset": {
        "name": "process",
        "period": 10000
    },
    "process": {
        "args": [
            "/tmp/go-build2159656503/b001/process.test",
            "-test.paniconexit0",
            "-test.timeout=10m0s",
            "-test.v=true",
            "-data",
            "-test.run=TestData"
        ],
        "command_line": "/tmp/go-build2159656503/b001/process.test -test.paniconexit0 -test.timeout=10m0s -test.v=true -data -test.run=TestData",
        "cpu": {
            "pct": 0.0012,
            "start_time": "2023-11-28T03:13:18.000Z"
        },
        "executable": "/tmp/go-build2159656503/b001/process.test",
        "memory": {
            "pct": 0.0008
        },
        "name": "process.test",
        "parent": {
            "pid": 592387
        },
        "pgid": 592387,
        "pid": 592516,
        "state": "sleeping",
        "working_directory": "/home/alexk/go/src/github.com/elastic/beats/metricbeat/module/system/process"
    },
    "service": {
        "type": "system"
    },
    "system": {
        "process": {
            "cgroup": {
                "cgroups_version": 2,
                "cpu": {
                    "id": "session-426.scope",
                    "path": "/user.slice/user-1000.slice/session-426.scope",
                    "pressure": {
                        "full": {
                            "10": {
                                "pct": 0
                            },
                            "300": {
                                "pct": 0
                            },
                            "60": {
                                "pct": 0
                            },
                            "total": 5524742
                        },
                        "some": {
                            "10": {
                                "pct": 0.07
                            },
                            "300": {
                                "pct": 0.1
                            },
                            "60": {
                                "pct": 0.3
                            },
                            "total": 32365561
                        }
                    },
                    "stats": {
                        "periods": 0,
                        "system": {
                            "norm": {
                                "pct": 0
                            },
                            "ns": 548263994,
                            "pct": 0
                        },
                        "throttled": {
                            "periods": 0,
                            "us": 0
                        },
                        "usage": {
                            "norm": {
                                "pct": 0
                            },
                            "ns": 1599791233,
                            "pct": 0
                        },
                        "user": {
                            "norm": {
                                "pct": 0
                            },
                            "ns": 1051527238,
                            "pct": 0
                        }
                    }
                },
                "id": "session-426.scope",
                "memory": {
                    "id": "session-426.scope",
                    "mem": {
                        "events": {
                            "high": 0,
                            "low": 0,
                            "max": 0,
                            "oom": 0,
                            "oom_kill": 0
                        },
                        "low": {
                            "bytes": 0
                        },
                        "usage": {
                            "bytes": 3864518656
                        }
                    },
                    "memsw": {
                        "events": {
                            "fail": 0,
                            "high": 0,
                            "max": 0
                        },
                        "low": {
                            "bytes": 0
                        },
                        "usage": {
                            "bytes": 0
                        }
                    },
                    "path": "/user.slice/user-1000.slice/session-426.scope",
                    "stats": {
                        "active_anon": {
                            "bytes": 1759969280
                        },
                        "active_file": {
                            "bytes": 990560256
                        },
                        "anon": {
                            "bytes": 1781649408
                        },
                        "anon_thp": {
                            "bytes": 618659840
                        },
                        "file": {
                            "bytes": 1710731264
                        },
                        "file_dirty": {
                            "bytes": 0
                        },
                        "file_mapped": {
                            "bytes": 15060992
                        },
                        "file_thp": {
                            "bytes": 0
                        },
                        "file_writeback": {
                            "bytes": 0
                        },
                        "htp_collapse_alloc": 313,
                        "inactive_anon": {
                            "bytes": 327753728
                        },
                        "inactive_file": {
                            "bytes": 698679296
                        },
                        "kernel_stack": {
                            "bytes": 2899968
                        },
                        "major_page_faults": 3001,
                        "page_activate": 0,
                        "page_deactivate": 0,
                        "page_faults": 79495294,
                        "page_lazy_free": 0,
                        "page_lazy_freed": 0,
                        "page_refill": 0,
                        "page_scan": 0,
                        "page_steal": 0,
                        "page_tables": {
                            "bytes": 19267584
                        },
                        "per_cpu": {
                            "bytes": 10336
                        },
                        "shmem": {
                            "bytes": 21491712
                        },
                        "shmem_thp": {
                            "bytes": 0
                        },
                        "slab": {
                            "bytes": 60957576
                        },
                        "slab_reclaimable": {
                            "bytes": 55816376
                        },
                        "slab_unreclaimable": {
                            "bytes": 5141200
                        },
                        "sock": {
                            "bytes": 0
                        },
                        "swap_cached": {
                            "bytes": 0
                        },
                        "thp_fault_alloc": 8577,
                        "unevictable": {
                            "bytes": 0
                        },
                        "workingset_activate_anon": 0,
                        "workingset_activate_file": 0,
                        "workingset_node_reclaim": 0,
                        "workingset_refault_anon": 0,
                        "workingset_refault_file": 0,
                        "workingset_restore_anon": 0,
                        "workingset_restore_file": 0
                    }
                },
                "path": "/user.slice/user-1000.slice/session-426.scope"
            },
            "cmdline": "/tmp/go-build2159656503/b001/process.test -test.paniconexit0 -test.timeout=10m0s -test.v=true -data -test.run=TestData",
            "cpu": {
                "start_time": "2023-11-28T03:13:18.000Z",
                "system": {
                    "ticks": 40
                },
                "total": {
                    "norm": {
                        "pct": 0.0012
                    },
                    "pct": 0.007,
                    "ticks": 100,
                    "value": 100
                },
                "user": {
                    "ticks": 60
                }
            },
            "fd": {
                "limit": {
                    "hard": 524288,
                    "soft": 524288
                },
                "open": 15
            },
            "io": {
                "cancelled_write_bytes": 0,
                "read_bytes": 0,
                "read_char": 2517537,
                "read_ops": 9551,
                "write_bytes": 0,
                "write_char": 22,
                "write_ops": 4
            },
            "memory": {
                "rss": {
                    "bytes": 26234880,
                    "pct": 0.0008
                },
                "share": 16252928,
                "size": 1886003200
            },
            "num_threads": 9,
            "state": "sleeping"
        }
    },
    "user": {
        "name": "alexk"
    }
}
```


