---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/freebsd-no-such-file.html
---

# open /compat/linux/proc: no such file or directory error on FreeBSD [freebsd-no-such-file]

The system metricsets rely on a Linux compatibility layer to retrieve metrics on FreeBSD. You need to mount the Linux procfs filesystem using the following commands. You may want to add these filesystems to your `/etc/fstab` so they are mounted automatically.

```sh
sudo mount -t procfs proc /proc
sudo mkdir -p /compat/linux/proc
sudo mount -t linprocfs /dev/null /compat/linux/proc
```

