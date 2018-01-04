{{ if .reference -}}
# The file integrity module sends events when files are changed (created,
# updated, deleted). The events contain file metadata and hashes.
{{ end -}}
- module: file_integrity
  {{ if eq .goos "darwin" -}}
  paths:
  - /bin
  - /usr/bin
  - /usr/local/bin
  - /sbin
  - /usr/sbin
  - /usr/local/sbin
  {{ else if eq .goos "windows" -}}
  paths:
  - C:/windows
  - C:/windows/system32
  - C:/Program Files
  - C:/Program Files (x86)
  {{ else -}}
  paths:
  - /bin
  - /usr/bin
  - /sbin
  - /usr/sbin
  - /etc
  {{- end }}
{{ if .reference }}
  # List of regular expressions to filter out notifications for unwanted files.
  # Wrap in single quotes to workaround YAML escaping rules. By default no files
  # are ignored.
  {{ if eq .goos "darwin" -}}
  exclude_files:
  - '\.DS_Store$'
  - '\.swp$'
  {{ else if eq .goos "windows" -}}
  exclude_files:
  - '(?i)\.lnk$'
  - '(?i)\.swp$'
  {{ else -}}
  exclude_files:
  - '(?i)\.sw[nop]$'
  - '~$'
  - '/\.git($|/)'
  {{- end }}

  # Scan over the configured file paths at startup and send events for new or
  # modified files since the last time Auditbeat was running.
  scan_at_start: true

  # Average scan rate. This throttles the amount of CPU and I/O that Auditbeat
  # consumes at startup while scanning. Default is "50 MiB".
  scan_rate_per_sec: 50 MiB

  # Limit on the size of files that will be hashed. Default is "100 MiB".
  # Limit on the size of files that will be hashed. Default is "100 MiB".
  max_file_size: 100 MiB

  # Hash types to compute when the file changes. Supported types are
  # blake2b_256, blake2b_384, blake2b_512, md5, sha1, sha224, sha256, sha384,
  # sha512, sha512_224, sha512_256, sha3_224, sha3_256, sha3_384 and sha3_512.
  # Default is sha1.
  hash_types: [sha1]

  # Detect changes to files included in subdirectories. Disabled by default.
  recursive: false
{{- end }}
