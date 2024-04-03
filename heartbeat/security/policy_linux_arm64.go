// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package security

import (
	"github.com/elastic/beats/v7/libbeat/common/seccomp"
	seccomp_types "github.com/elastic/go-seccomp-bpf"
)

func init() {
	// Register deny-by-default based policy for ARM platforms
	defaultPolicy := &seccomp_types.Policy{
		DefaultAction: seccomp_types.ActionErrno,
		Syscalls: []seccomp_types.SyscallGroup{
			{
				Action: seccomp_types.ActionAllow,
				Names: []string{
					"accept",
					"accept4",
					"bind",
					"brk",
					"capget",
					"capset",
					"chdir",
					"clock_gettime",
					"clone",
					"clone3",
					"close",
					"connect",
					"dup",
					"dup3",
					"epoll_create1",
					"epoll_ctl",
					"epoll_pwait",
					"eventfd2",
					"execve",
					"exit",
					"exit_group",
					"faccessat",
					"fadvise64",
					"fallocate",
					"fchdir",
					"fchmod",
					"fchmodat",
					"fchown",
					"fchownat",
					"fcntl",
					"fdatasync",
					"flock",
					"fstat",
					"fstatat", // or newfstatat
					"fstatfs",
					"fsync",
					"ftruncate",
					"futex",
					"getcwd",
					"getdents64",
					"getegid",
					"geteuid",
					"getgid",
					"getgroups",
					"getpeername",
					"getpgid",
					"getpid",
					"getppid",
					"getpriority",
					"getrandom",
					"getresgid",
					"getresuid",
					"getrlimit",
					"getrusage",
					"getsockname",
					"getsockopt",
					"gettid",
					"gettimeofday",
					"getuid",
					"inotify_add_watch",
					"inotify_init1",
					"inotify_rm_watch",
					"ioctl",
					"kill",
					"linkat",
					"listen",
					"lseek",
					"madvise",
					"memfd_create",
					"mincore",
					"mkdirat",
					"mlock",
					"mmap",
					"mprotect",
					"munmap",
					"name_to_handle_at",
					"nanosleep",
					"openat",
					"pipe2",
					"ppoll",
					"prctl",
					"pread64",
					"prlimit64",
					"pselect6",
					"pwrite64",
					"read",
					"readlinkat",
					"recvfrom",
					"recvmmsg",
					"recvmsg",
					"renameat",
					"rseq",
					"rt_sigaction",
					"rt_sigprocmask",
					"rt_sigreturn",
					"sched_getaffinity",
					"sched_getattr",
					"sched_getparam",
					"sched_getscheduler",
					"sched_setaffinity",
					"sched_setattr",
					"sched_yield",
					"seccomp",
					"sendfile",
					"sendmmsg",
					"sendmsg",
					"sendto",
					"set_robust_list",
					"set_tid_address",
					"setgid",
					"setgroups",
					"setitimer",
					"setpriority",
					"setsid",
					"setsockopt",
					"setuid",
					"shutdown",
					"sigaltstack",
					"socket",
					"socketpair",
					"splice",
					"statfs",
					"statx",
					"symlinkat",
					"sysinfo",
					"tgkill",
					"tkill",
					"umask",
					"uname",
					"unlinkat",
					"utimensat",
					"wait4",
					"waitid",
					"write",
					"writev",
				},
			},
		},
	}

	seccomp.MustRegisterPolicy(defaultPolicy)
}
