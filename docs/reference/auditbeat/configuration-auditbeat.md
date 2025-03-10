---
navigation_title: "Modules"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/configuration-auditbeat.html
---

# Configure modules [configuration-auditbeat]


To enable specific modules you add entries to the `auditbeat.modules` list in the `auditbeat.yml` config file. Each entry in the list begins with a dash (-) and is followed by settings for that module.

The following example shows a configuration that runs the `auditd` and `file_integrity` modules.

```yaml
auditbeat.modules:

- module: auditd
  audit_rules: |
    -w /etc/passwd -p wa -k identity
    -a always,exit -F arch=b32 -S open,creat,truncate,ftruncate,openat,open_by_handle_at -F exit=-EPERM -k access

- module: file_integrity
  paths:
  - /bin
  - /usr/bin
  - /sbin
  - /usr/sbin
  - /etc
```

The configuration details vary by module. See the [module documentation](/reference/auditbeat/auditbeat-modules.md) for more detail about configuring the available modules.

