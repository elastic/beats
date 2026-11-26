## 9.5.7 [beats-9.5.7-breaking-changes]

**Auditbeat**

::::{dropdown} The `file_integrity` `ebpf` backend requires the host PID namespace when run in Docker/Kubernetes.
The `file_integrity` module's `ebpf` backend now sources file events through quark instead of the `ebpfevents` library, removing the dependency on `elastic/ebpfevents` and `elastic/ebpf`. Configuration is unchanged. Each operation is still reported as its own event, and `file.mode` now reports only the permission bits, consistent with the `fsnotify` backend.

quark attributes each event to a process by looking up the process ID the kernel reports in a table built from `/proc` and kept current from kernel events. The kernel reports host process IDs. When Auditbeat runs in a Docker container or Kubernetes pod with its own PID namespace, that table is built from the container's process IDs instead.

For more information, check [#53507](https://github.com/elastic/beats/pull/53507) [#22396](https://github.com/elastic/beats/issues/22396).

**Impact**<br>Without `--pid=host` or `hostPID: true`, file events from processes that were already running when Auditbeat started are dropped, and events from processes started later can be attributed to the wrong process when a host process ID happens to match a container one. The `system/process` dataset and the `add_session_metadata` processor with kernel tracing already have this requirement. Deployments outside Docker/Kubernetes, and the `fsnotify` and `kprobes` backends, are unaffected.

**Action**<br>Add `--pid=host` to the `docker run` command, alongside the existing `--cap-add=SYS_ADMIN --cap-add=BPF` and the `/sys` bind mount. In Kubernetes, set `hostPID: true` on the pod.
::::
