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
	switch runtime.GOARCH {
	case "amd64", "386":
		if localUserName := os.Getenv("BEAT_LOCAL_USER"); localUserName != "" && syscall.Geteuid() == 0 {
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
			err = syscall.Setgroups([]int{localUserUid, 0})
			if err != nil {
				panic(fmt.Sprintf("could not prsetgroups: %s", err))
			}

			err = syscall.Setgid(1000)
			if err != nil {
				panic(fmt.Sprintf("could not set gid to %d: %s", localUserGid, err))
			}

			// Note this is not the regular SetUID! Look at the package docs for it, it preserves
			// capabilities post-SetUID, which we use to lock things down immediately
			err = cap.SetUID(localUserUid)
			if err != nil {
				panic(fmt.Sprintf("could not setuid to %d: %s", localUserUid, err))
			}

			u, err := user.Lookup(localUserName)
			if err != nil {
				panic(fmt.Sprintf("could not lookup local username: %s", err))
			}
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

			newcaps.SetProc()
			if err != nil {
				panic(fmt.Sprintf("error setting new process capabilities via setcap: %s", err))
			}
			groups, _ := syscall.Getgroups()
			logp.Info("Now running as uid: %d, gid: %d, with groups: %v, and with capabilities: %s\n", syscall.Geteuid(), syscall.Getegid(), groups, cap.GetProc())
		}

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
			"clone",
			"chdir",
			"close",
			"epoll_ctl",
			"epoll_pwait",
			"execve",
			"exit",
			"fcntl",
			"flock",
			"fstat",
			"futex",
			"geteuid",
			"getegid",
			"getgid",
			"getpid",
			"getppid",
			"getrandom",
			"getsockname",
			"gettid",
			"getuid",
			"ioctl",
			"mlock",
			"mmap",
			"mprotect",
			"munmap",
			"newfstatat",
			"openat",
			"eventfd2",
			"prctl",
			"mkdtemp",
			"pread64",
			"prlimit64",
			"read",
			"readlinkat",
			"recvfrom",
			"rt_sigaction",
			"rt_sigprocmask",
			"rt_sigreturn",
			"sched_getaffinity",
			"sendto",
			"set_robust_list",
			"set_tid_address",
			"sigaltstack",
			"socket",
			"umask",
			"uname",
			"write",
			"eouueouoe",
		}

		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, syscalls...); err != nil {
			panic(err)
		}
		fmt.Printf("Installed seccomp policy")
	}
}
