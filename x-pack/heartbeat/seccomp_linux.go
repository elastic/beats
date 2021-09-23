// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"runtime"

	"github.com/elastic/beats/v7/libbeat/common/seccomp"
)

func init() {
	switch runtime.GOARCH {
	case "amd64", "386":
		// We require a number of syscalls to run. This list was generated with
		// mage build && env ELASTIC_SYNTHETICS_CAPABLE=true strace --output=syscalls  ./heartbeat --path.config sample-synthetics-config/ -e
		// then filtered through:  cat syscalls | cut -d '(' -f 1 | egrep '\w+' -o | sort | uniq | xargs -n1 -IFF echo \"FF\"
		// We should tighten this up before GA. While it is true that there are probably duplicate
		// syscalls here vs. the base, this is probably OK for now.
		syscalls := []string{
			"access",
			"arch_prctl",
			"bind",
			"brk",
			"capget",
			"chdir",
			"chmod",
			"chown",
			"clone",
			"close",
			"connect",
			"creat",
			"dup2",
			"epoll_ctl",
			"epoll_pwait",
			"eventfd2",
			"execve",
			"exit",
			"faccessat",
			"fadvise64",
			"fallocate",
			"fcntl",
			"flock",
			"fstat",
			"fsync",
			"futex",
			"capget",
			"getcwd",
			"getdents64",
			"getegid",
			"geteuid",
			"getgid",
			"getpeername",
			"getpgrp",
			"getpid",
			"getppid",
			"getpriority",
			"getrandom",
			"getresuid",
			"getresgid",
			"getrusage",
			"getsockname",
			"gettid",
			"getuid",
			"ioctl",
			"inotify_init",
			"lchown",
			"link",
			"lseek",
			"madvise",
			"memfd_create",
			"mkdir",
			"mkdirat",
			"mlock",
			"mmap",
			"mprotect",
			"munmap",
			"nanosleep",
			"name_to_handle_at",
			"newfstatat",
			"openat",
			"pipe",
			"pipe2",
			"poll",
			"prctl",
			"pread64",
			"prlimit64",
			"pwrite64",
			"read",
			"readlink",
			"readlinkat",
			"recvfrom",
			"rename",
			"rmdir",
			"rt_sigaction",
			"rt_sigprocmask",
			"rt_sigreturn",
			"sched_getaffinity",
			"sched_getparam",
			"sched_getscheduler",
			"select",
			"sendto",
			"set_robust_list",
			"set_tid_address",
			"setpriority",
			"setsid",
			"sigaltstack",
			"socket",
			"socketpair",
			"stat",
			"statx",
			"symlink",
			"umask",
			"uname",
			"unlink",
			"utimensat",
			"write",
		}

		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, syscalls...); err != nil {
			panic(err)
		}
	}
}
