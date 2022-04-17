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
	"github.com/menderesk/go-seccomp-bpf"
)

func init() {
	defaultPolicy = &seccomp.Policy{
		DefaultAction: seccomp.ActionErrno,
		Syscalls: []seccomp.SyscallGroup{
			{
				Action: seccomp.ActionAllow,
				Names: []string{
					"_llseek",
					"access",
					"brk",
					"chmod",
					"chown",
					"clock_gettime",
					"clone",
					"clone3",
					"close",
					"dup",
					"dup2",
					"epoll_create",
					"epoll_create1",
					"epoll_ctl",
					"epoll_wait",
					"exit",
					"exit_group",
					"fchdir",
					"fchmod",
					"fchmodat",
					"fchown32",
					"fchownat",
					"fcntl",
					"fcntl64",
					"fdatasync",
					"flock",
					"fstat64",
					"fstatat64",
					"fsync",
					"ftruncate64",
					"futex",
					"getcwd",
					"getdents",
					"getdents64",
					"geteuid32",
					"getgid32",
					"getpid",
					"getppid",
					"getrandom",
					"getrlimit",
					"getrusage",
					"gettid",
					"gettimeofday",
					"getuid32",
					"inotify_add_watch",
					"inotify_init1",
					"inotify_rm_watch",
					"ioctl",
					"kill",
					"lstat64",
					"madvise",
					"mincore",
					"mkdirat",
					"mmap2",
					"mprotect",
					"munmap",
					"nanosleep",
					"open",
					"openat",
					"pipe",
					"pipe2",
					"poll",
					"pread64",
					"prlimit64",
					"pselect6",
					"pwrite64",
					"read",
					"readlink",
					"readlinkat",
					"rename",
					"renameat",
					"restart_syscall",
					"rseq",
					"rt_sigaction",
					"rt_sigprocmask",
					"rt_sigreturn",
					"sched_getaffinity",
					"sched_yield",
					"sendfile64",
					"set_robust_list",
					"set_thread_area",
					"setgid32",
					"setgroups32",
					"setitimer",
					"setuid32",
					"sigaltstack",
					"socketcall",
					"splice",
					"stat",
					"stat64",
					"statfs64",
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
