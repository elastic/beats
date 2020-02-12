# Package seccomp

Package seccomp loads a Linux seccomp BPF filter that controls what system calls
the process can use. Here is a description of the files in this directory.

| File                           | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
|--------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| seccomp.go                     | Contains the code for registering application specific policies and for loading the seccomp BPF filter.                                                                                                                                                                                                                                                                                                                                                                          |
| policy.go.tpl                  | This is a Go text/template used to generate code for a whitelist seccomp BPF policy. The template is used when running `make seccomp` and `make seccomp-package`. The tool that reads the template file is the [seccomp-profiler](https://github.com/elastic/go-seccomp-bpf/tree/master/cmd/seccomp-profiler).                                                                                                                                                                     |
| seccomp-profiler-blacklist.txt | A list of system calls that are filtered from the resulting policy even if they are detected as being used in a binary. For example if the os/exec package is included by a dependency it will cause several system calls to be included such as execve, ptrace, and chroot. The system calls are never actually used so we want them removed from the profile.                                                                                                                  |
| seccomp-profiler-allow.txt     | A list of system calls that are always included in the resulting policy. System calls that are made from CGO or linked libraries (such as libpthread) are not detected by seccomp-profiler and must be manually added to the list. Tools like `strace -c` and Auditbeat (where event.action=violated-seccomp-policy) can be used to find missing system calls. System calls from any architecture may be added to the list because it is automatically filtered based on architecture. |
| policy_linux_$arch.go          | This is a default policy provided by libbeat. It is used when a Beat does not register its own application specific policy with `seccomp.MustRegisterPolicy()` (e.g. community Beats).|

## Developing a Whitelist Policy

Policy files can be generated for amd64 and 386 architectures (our tooling is
currently limited to these architectures). You can generate a policy by building
the packages and then profiling the resulting binaries. This ensures that the
profiles are built based on the binaries you plan to release. The policies are
stored at `$beatname/include/seccomp_linux_$goarch.go`.

```sh
make release && make seccomp-package
```

If you are developing on Linux you can profile the binary produced by `make`.
It will only generate a profile for the current architecture.

```sh
make && make seccomp
```

You should thoroughly test the policy as described below and add any additional
syscalls that are detected.

The `include` package must be imported by the Beat such that the policy is
registered as a side-effect.

```go
import (
    _ "github.com/user/beatname/include"
)
```

## Testing Seccomp Policies

### Linux Auditing

Seccomp violations are reported by Linux kernel's audit subsystem. So you can run
Auditbeat to look for system call denials. With this config Auditbeat will
report only seccomp events.

```yaml
auditbeat.modules:
- module: auditd
  processors:
  - drop_event.when.not.equals.event.type: seccomp

output.console.pretty: true

logging.level: warning
```

Then start Auditbeat and it will log events to stdout.

`sudo auditbeat -e`

### strace

Another profiling option is `strace -c`. This will record all system calls used
by the Beat. Then this list can be used to steer the development of a policy.

```sh
sudo strace -c ./metricbeat -strict.perms=false
% time     seconds  usecs/call     calls    errors syscall
------ ----------- ----------- --------- --------- ----------------
 79.75    0.002521          97        26           futex
  2.97    0.000094           3        34           mmap
  2.69    0.000085           1       120           rt_sigaction
  2.25    0.000071          18         4           open
  1.96    0.000062          12         5         1 openat
  1.68    0.000053           5        11           read
  1.46    0.000046           4        12           getrandom
  1.42    0.000045           4        11           mprotect
  1.08    0.000034           6         6           fstat
  0.85    0.000027          14         2           munmap
  0.70    0.000022           6         4           clone
  0.57    0.000018          18         1           ioctl
  0.47    0.000015           2         8           close
  0.41    0.000013           2         8         2 epoll_ctl
  0.38    0.000012           1        11           rt_sigprocmask
  0.22    0.000007           7         1           newfstatat
  0.19    0.000006           6         1           write
  0.16    0.000005           1         6           fcntl
  0.13    0.000004           1         3           brk
  0.13    0.000004           1         5         5 access
  0.13    0.000004           4         1           sched_getaffinity
  0.09    0.000003           2         2           sigaltstack
  0.06    0.000002           2         1           getcwd
  0.06    0.000002           2         1           getrlimit
  0.06    0.000002           2         1           arch_prctl
  0.06    0.000002           2         1           gettid
  0.03    0.000001           1         1           set_tid_address
  0.03    0.000001           1         1           set_robust_list
  0.00    0.000000           0         1           execve
  0.00    0.000000           0         1           getuid
  0.00    0.000000           0         1           getgid
  0.00    0.000000           0         1           readlinkat
  0.00    0.000000           0         1           epoll_create1
------ ----------- ----------- --------- --------- ----------------
100.00    0.003161                   293         8 total
```
