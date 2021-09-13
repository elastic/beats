// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"syscall"

	"github.com/elastic/beats/v7/libbeat/common/seccomp"
	"github.com/elastic/beats/v7/libbeat/logp"
	"kernel.org/pub/linux/libs/security/libcap/cap"
)

func init() {
	if localUserName := os.Getenv("BEAT_LOCAL_USER"); localUserName != "" && syscall.Geteuid() == 0 && runtime.GOOS == "linux" {
		localUser, err := user.Lookup(localUserName)
		if err != nil {
			panic(fmt.Sprintf("could not lookup BEAT_LOCAL_USER=%s: %s", localUser, err))
		}
		localUserUid, err := strconv.Atoi(localUser.Uid)
		if err != nil {
			panic(fmt.Sprintf("could not parse UID '%s' as int: %s", localUser.Uid, err))
		}
		localUserGid, err := strconv.Atoi(localUser.Gid)
		if err != nil {
			panic(fmt.Sprintf("could not parse GID '%s' as int: %s", localUser.Uid, err))
		}
		// We include the root group because the docker image contains many directories (data,logs)
		// that are owned by root:root with 0775 perms. The heartbeat user is in both groups
		// in the container, but we need to repeat that here.
		err = syscall.Setgroups([]int{localUserUid, 0})
		if err != nil {
			panic(fmt.Sprintf("could not prsetgroups: %s", err))
		}

		// Set the main group as 1000 so new files created are owned by the user's group
		err = syscall.Setgid(1000)
		if err != nil {
			panic(fmt.Sprintf("could not set gid to %d: %s", localUserGid, err))
		}

		// Note this is not the regular SetUID! Look at the 'cap' package docs for it, it preserves
		// capabilities post-SetUID, which we use to lock things down immediately
		err = cap.SetUID(localUserUid)
		if err != nil {
			panic(fmt.Sprintf("could not setuid to %d: %s", localUserUid, err))
		}

		u, err := user.Lookup(localUserName)
		if err != nil {
			panic(fmt.Sprintf("could not lookup local username: %s", err))
		}

		// This may not be necessary, but is good hygeine, we do some shelling out to node/npm etc.
		// and $HOME should reflect the user's preferences
		os.Setenv("HOME", u.HomeDir)

		// Start with an empty capability set
		newcaps := cap.NewSet()
		// Both permitted and effective are required! Permitted makes the permmission
		// possible to get, effective makes it 'active'
		err = newcaps.SetFlag(cap.Permitted, true, cap.NET_RAW)
		if err != nil {
			panic(fmt.Sprintf("error setting permitted setcap: %s", err))
		}
		err = newcaps.SetFlag(cap.Effective, true, cap.NET_RAW)
		if err != nil {
			panic(fmt.Sprintf("error setting effective setcap: %s", err))
		}

		// We do not want these capabilities to be inherited by subprocesses
		err = newcaps.SetFlag(cap.Inheritable, false, cap.NET_RAW)
		if err != nil {
			panic(fmt.Sprintf("error setting inheritable setcap: %s", err))
		}

		// Apply the new capabilities to the current process (incl. all threads)
		newcaps.SetProc()
		if err != nil {
			panic(fmt.Sprintf("error setting new process capabilities via setcap: %s", err))
		}

		groups, _ := syscall.Getgroups()
		logp.Info("Now running as '%s' with effect user/group ids %d/%d, with groups: %v, and with capabilities: %s\n", localUser, syscall.Geteuid(), syscall.Getegid(), groups, cap.GetProc())
	}

	switch runtime.GOARCH {
	case "amd64", "386":
		// We require a number of syscalls to run. This list was generated with
		// mage build && env ELASTIC_SYNTHETICS_CAPABLE=true strace -f --output=syscalls  ./heartbeat --path.config sample-synthetics-config/ -e
		// then grepping for 'EPERM' in the 'syscalls' file.
		syscalls := []string{
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
		logp.Info("Installed seccomp policy")
	}
}
