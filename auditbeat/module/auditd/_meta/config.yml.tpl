{{ if eq .goos "linux" -}}
{{ if .reference -}}
# The auditd module collects events from the audit framework in the Linux
# kernel. You need to specify audit rules for the events that you want to audit.
{{ end -}}
- module: auditd
  {{ if .reference -}}
  resolve_ids: true
  failure_mode: silent
  backlog_limit: 8196
  rate_limit: 0
  include_raw_message: false
  include_warnings: false
  {{ end -}}
  audit_rules: |
    ## Define audit rules here.
    ## Create file watches (-w) or syscall audits (-a or -A). Uncomment these
    ## examples or add your own rules.

    {{ if eq .goarch "amd64" -}}
    ## If you are on a 64 bit platform, everything should be running
    ## in 64 bit mode. This rule will detect any use of the 32 bit syscalls
    ## because this might be a sign of someone exploiting a hole in the 32
    ## bit API.
    #-a always,exit -F arch=b32 -S all -F key=32bit-abi

    {{ end -}}
    ## Executions.
    #-a always,exit -F arch=b{{.arch_bits}} -S execve,execveat -k exec

    ## External access (warning: these can be expensive to audit).
    #-a always,exit -F arch=b{{.arch_bits}} -S accept,bind,connect -F key=external-access

    ## Identity changes.
    #-w /etc/group -p wa -k identity
    #-w /etc/passwd -p wa -k identity
    #-w /etc/gshadow -p wa -k identity

    ## Unauthorized access attempts.
    #-a always,exit -F arch=b{{.arch_bits}} -S open,creat,truncate,ftruncate,openat,open_by_handle_at -F exit=-EACCES -k access
    #-a always,exit -F arch=b{{.arch_bits}} -S open,creat,truncate,ftruncate,openat,open_by_handle_at -F exit=-EPERM -k access
{{ end -}}
