---
navigation_title: "Known issues"

---

# Beats known issues [beats-known-issues]

Known issues are significant defects or limitations that may impact your implementation. These issues are actively being worked on and will be addressed in a future release. Review known issues to help you make informed decisions, such as upgrading to a new version.

% Use the following template to add entries to this page.

% :::{dropdown} Title of known issue
% **Details** 
% On [Month/Day/Year], a known issue was discovered that [description of known issue].

% **Workaround** 
% Workaround description.

% **Resolved**
% On [Month/Day/Year], this issue was resolved.

% :::

:::{dropdown} Winlogbeat and Filebeat `winlog` input can crash the Event Log on Windows Server 2025.
**Details** 
On 04/16/2025, a known issue was discovered that can cause a crash of the Event Log service in Windows Server 2025 **when reading forwarded events in an Event Collector setup**. The issue appears for some combinations of filters where the OS handles non-null-terminated strings, leading to the crash.

**Workaround** 
As a workaround, and to prevent crashes, Beats will ignore any filters provided when working with forwarded events on Windows Server 2025 until the issue is resolved.

% **Resolved**
% On [Month/Day/Year], this issue was resolved.
:::


:::{dropdown} Filebeat's Filestream input does not validate `clean_inactive`.
**Applies to**: Filebeat < 9.2.0

The Filestream input does not enforce the restrictions documented for
the `clean_inactive` option, thus allowing configurations that can
lead to data re-ingestion issues.

**Fixed planned in**: 9.2.0 by [PR #46373](https://github.com/elastic/beats/pull/46373)
:::

:::{dropdown} Setting `clean_inactive` to `0` in Filebeat's Filestream input will cause data to be re-ingested on every restart.
**Applies to**: Filebeat >= 8.14.0 and < 9.2.0

When `clean_inactive` is set to `0`, Filestream will clean the state of all files
on start up, effectively re-ingesting all files on restart.

**Workaround**
- For Filestream >= 8.16.0 and < 9.2.0: disable `clean_inactive` by setting `clean_inactive: -1`.
- For Filestream >= 8.14.0 and < 8.16.0 set `clean_inactive` to a very
large value. For example, use `clean_inactive: 43800h0m0s`, which is 5 years.

**Fixed planned in**: 9.2.0 by [PR #46373](https://github.com/elastic/beats/pull/46373)
:::

:::{dropdown} Beats panic on restart when "restart_on_cert_change" is enabled on Linux

**Applies to**: v8.16.6+, v8.17.3+, v8.18.0+, v8.19.0+, v9.0.0+, and v9.1.0+

**Details**
A known issue was discovered where Beats running on Linux with `restart_on_cert_change` enabled panic during a restart. This occurs because the default seccomp policy does not include the `eventfd2` syscall, which is used by Go runtime versions 1.23.0. While the initial launch is successful, subsequent restarts fail as the seccomp policy is already active, blocking the required syscall.

**Workaround**
Add a custom seccomp policy to the beat configuration file that explicitly includes the eventfd2 syscall. This custom policy overrides the default, so it must contain a complete list of all required syscalls.
```
seccomp:
  syscalls:
    - action: allow
      names:
        - accept
        - accept4
        - access
        - arch_prctl
        - bind
        - brk
        - capget
        - chmod
        - chown
        - clock_gettime
        - clock_nanosleep
        - clone
        - clone3
        - close
        - connect
        - dup
        - dup2
        - dup3
        - epoll_create
        - epoll_create1
        - epoll_ctl
        - epoll_pwait
        - epoll_wait
        - eventfd2
        - execve
        - exit
        - exit_group
        - faccessat
        - faccessat2
        - fchdir
        - fchmod
        - fchmodat
        - fchown
        - fchownat
        - fcntl
        - fdatasync
        - flock
        - fstat
        - fstatfs
        - fsync
        - ftruncate
        - futex
        - getcwd
        - getdents
        - getdents64
        - geteuid
        - getgid
        - getpeername
        - getpid
        - getppid
        - getrandom
        - getrlimit
        - getrusage
        - getsockname
        - getsockopt
        - gettid
        - gettimeofday
        - getuid
        - inotify_add_watch
        - inotify_init1
        - inotify_rm_watch
        - ioctl
        - kill
        - listen
        - lseek
        - lstat
        - madvise
        - mincore
        - mkdirat
        - mmap
        - mprotect
        - munmap
        - nanosleep
        - newfstatat
        - open
        - openat
        - pipe
        - pipe2
        - poll
        - ppoll
        - prctl
        - pread64
        - pselect6
        - pwrite64
        - read
        - readlink
        - readlinkat
        - recvfrom
        - recvmmsg
        - recvmsg
        - rename
        - renameat
        - rseq
        - rt_sigaction
        - rt_sigprocmask
        - rt_sigreturn
        - sched_getaffinity
        - sched_yield
        - sendfile
        - sendmmsg
        - sendmsg
        - sendto
        - set_robust_list
        - setitimer
        - setrlimit
        - setsockopt
        - shutdown
        - sigaltstack
        - socket
        - splice
        - stat
        - statfs
        - sysinfo
        - tgkill
        - time
        - tkill
        - uname
        - unlink
        - unlinkat
        - wait4
        - waitid
        - write
        - writev
```

% **Resolved**
% This issue was resolved by updating the default seccomp policy to include the `eventfd2` syscall. To apply the fix, please upgrade to version 8.18.7, 8.19.4, 9.0.7, 9.1.4, or any subsequent release.
:::
