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
    # Define audit rules here.
    # Create file watches (-w) or syscall audits (-a or -A). For example:
    #-w /etc/passwd -p wa -k identity
    #-a always,exit -F arch=b32 -S open,creat,truncate,ftruncate,openat,open_by_handle_at -F exit=-EPERM -k access

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
  # Limit on the size of files that will be hashed. Default is 100 MiB.
  file.max_file_size: 100 MiB

  # Hash types to compute when the file changes. Supported types are md5, sha1,
  # sha224, sha256, sha384, sha512, sha512_224, and sha512_256. Default is sha1.
  file.hash_types: [sha1]
  {{- end }}
