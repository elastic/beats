{{ if eq .goos "linux" -}}
{{ if .reference -}}
# The kernel metricset collects events from the audit framework in the Linux
# kernel. You need to specify audit rules for the events that you want to audit.
{{ end -}}
- module: audit
  metricsets: [kernel]
  {{ if .reference -}}
  kernel.resolve_ids: true
  kernel.failure_mode: silent
  kernel.backlog_limit: 8196
  kernel.rate_limit: 0
  kernel.include_raw_message: false
  kernel.include_warnings: false
  {{ end -}}
  kernel.audit_rules: |
    ## Define audit rules here.
    ## Create file watches (-w) or syscall audits (-a or -A). Uncomment these
    ## examples or add your own rules.

    ## If you are on a 64 bit platform, everything should be running
    ## in 64 bit mode. This rule will detect any use of the 32 bit syscalls
    ## because this might be a sign of someone exploiting a hole in the 32
    ## bit API.
    #-a always,exit -F arch=b32 -S all -F key=32bit-abi

    ## Executions.
    #-a always,exit -F arch=b64 -S execve,execveat -k exec

    ## External access.
    #-a always,exit -F arch=b64 -S accept,bind,connect,recvfrom -F key=external-access

    ## Identity changes.
    #-w /etc/group -p wa -k identity
    #-w /etc/passwd -p wa -k identity
    #-w /etc/gshadow -p wa -k identity

    ## Unauthorized access attempts.
    #-a always,exit -F arch=b64 -S open,creat,truncate,ftruncate,openat,open_by_handle_at -F exit=-EACCES -k access
    #-a always,exit -F arch=b64 -S open,creat,truncate,ftruncate,openat,open_by_handle_at -F exit=-EPERM -k access

{{ end -}}

{{ if .reference -}}
# The file integrity metricset sends events when files are changed (created,
# updated, deleted). The events contain file metadata and hashes.
{{ end -}}
- module: audit
  metricsets: [file]
  {{ if eq .goos "darwin" -}}
  file.paths:
  - /bin
  - /usr/bin
  - /usr/local/bin
  - /sbin
  - /usr/sbin
  - /usr/local/sbin
  {{ else if eq .goos "windows" -}}
  file.paths:
  - C:/windows
  - C:/windows/system32
  - C:/Program Files
  - C:/Program Files (x86)
  {{ else -}}
  file.paths:
  - /bin
  - /usr/bin
  - /sbin
  - /usr/sbin
  - /etc
  {{ end -}}
  {{ if .reference }}
  # Scan over the configured file paths at startup and send events for new or
  # modified files since the last time Auditbeat was running.
  file.scan_at_start: true

  # Average scan rate. This throttles the amount of CPU and I/O that Auditbeat
  # consumes at startup while scanning. Default is "50 MiB".
  file.scan_rate_per_sec: 50 MiB

  # Limit on the size of files that will be hashed. Default is "100 MiB".
  file.max_file_size: 100 MiB

  # Hash types to compute when the file changes. Supported types are md5, sha1,
  # sha224, sha256, sha384, sha512, sha512_224, sha512_256, sha3_224, sha3_256,
  # sha3_384 and sha3_512. Default is sha1.
  file.hash_types: [sha1]
  {{- end }}
