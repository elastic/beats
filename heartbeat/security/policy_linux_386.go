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

import "github.com/elastic/beats/v7/libbeat/common/seccomp"

func init() {
	var syscalls = []string{
		"access",
		"arch_prctl",
		"bind",
		"brk",
		"capget",
		"capset",
		"chdir",
		"chmod",
		"chown",
		"clone",
		"close",
		"connect",
		"creat",
		"dup",
		"dup2",
		"dup3",
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
		"getgroups",
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
		"setgid",
		"setgroups",
		"setpriority",
		"setsid",
		"setuid",
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
	err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, syscalls...)
	if err != nil {
		panic(err)
	}
}
