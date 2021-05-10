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

package seccomp

import (
	"github.com/elastic/go-seccomp-bpf"
)

func init() {
	defaultPolicy = &seccomp.Policy{
		DefaultAction: seccomp.ActionErrno,
		Syscalls: []seccomp.SyscallGroup{
			{
				Action: seccomp.ActionAllow,
				Names: []string{
					"accept",
					"accept4",
					"access",
					"arch_prctl",
					"bind",
					"brk",
					"chmod",
					"chown",
					"clock_gettime",
					"clone",
					"close",
					"connect",
					"dup",
					"dup2",
					"epoll_create",
					"epoll_create1",
					"epoll_ctl",
					"epoll_pwait",
					"epoll_wait",
					"exit",
					"exit_group",
					"fchdir",
					"fchmod",
					"fchmodat",
					"fchown",
					"fchownat",
					"fcntl",
					"fdatasync",
					"flock",
					"fstat",
					"fstatfs",
					"fsync",
					"ftruncate",
					"futex",
					"getcwd",
					"getdents",
					"getdents64",
					"geteuid",
					"getgid",
					"getpeername",
					"getpid",
					"getppid",
					"getrandom",
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
					"listen",
					"lseek",
					"lstat",
					"madvise",
					"mincore",
					"mkdirat",
					"mmap",
					"mprotect",
					"munmap",
					"nanosleep",
					"newfstatat",
					"open",
					"openat",
					"pipe",
					"pipe2",
					"poll",
					"ppoll",
					"pread64",
					"pselect6",
					"pwrite64",
					"read",
					"readlink",
					"readlinkat",
					"recvfrom",
					"recvmmsg",
					"recvmsg",
					"rename",
					"renameat",
					"rt_sigaction",
					"rt_sigprocmask",
					"rt_sigreturn",
					"sched_getaffinity",
					"sched_yield",
					"sendfile",
					"sendmmsg",
					"sendmsg",
					"sendto",
					"set_robust_list",
					"setitimer",
					"setsockopt",
					"shutdown",
					"sigaltstack",
					"socket",
					"splice",
					"stat",
					"statfs",
					"sysinfo",
					"tgkill",
					"time",
					"tkill",
					"uname",
					"unlink",
					"unlinkat",
					"wait4",
					"waitid",
					"write",
					"writev",
				},
			},
		},
	}
}
