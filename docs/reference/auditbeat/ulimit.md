---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/ulimit.html
---

# Auditbeat fails to watch folders because too many files are open [ulimit]

Because of the way file monitoring is implemented on macOS, you may see a warning similar to the following:

```shell
eventreader_fsnotify.go:42: WARN [audit.file] Failed to watch /usr/bin: too many
open files (check the max number of open files allowed with 'ulimit -a')
```

To resolve this issue, run Auditbeat with the `ulimit` set to a larger value, for example:

```sh
sudo sh -c 'ulimit -n 8192 && ./Auditbeat -e
```

Or:

```sh
sudo su
ulimit -n 8192
./auditbeat -e
```

