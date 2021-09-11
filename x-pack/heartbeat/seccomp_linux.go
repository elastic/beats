// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/beats/v7/libbeat/common/seccomp"
	"kernel.org/pub/linux/libs/security/libcap/cap"
)

func init() {
	switch runtime.GOARCH {
	case "amd64", "386":
		// We require a number of syscalls to run. This list was generated with
		// mage build && env ELASTIC_SYNTHETICS_CAPABLE=true strace --output=syscalls  ./heartbeat --path.config sample-synthetics-config/ -e
		// then filtered through:  cat syscalls | cut -d '(' -f 1 | egrep '\w+' -o | sort | uniq | xargs -n1 -IFF echo \"FF\"
		// We should tighten this up before GA. While it is true that there are probably duplicate
		// syscalls here vs. the base, this is probably OK for now.
		var err error
		if os.Getuid() == 0 {
			err = cap.SetUID(1000)
			if err != nil {
				panic(err)
			}

			newcaps := cap.NewSet()
			/*
				err = newcaps.SetFlag(cap.Effective, false, cap.CHOWN, cap.DAC_OVERRIDE, cap.DAC_READ_SEARCH, cap.FOWNER, cap.FSETID, cap.KILL, cap.SETGID, cap.SETUID, cap.SETPCAP, cap.LINUX_IMMUTABLE, cap.SYS_MODULE, cap.SYS_CHROOT, cap.SYS_PTRACE, cap.SYS_PACCT, cap.SYS_ADMIN, cap.SETUID)
				err = newcaps.SetFlag(cap.Effective, false, cap.CHOWN, cap.DAC_OVERRIDE, cap.DAC_READ_SEARCH, cap.FOWNER, cap.FSETID, cap.KILL, cap.SETGID, cap.SETUID, cap.SETPCAP, cap.LINUX_IMMUTABLE, cap.SYS_MODULE, cap.SYS_CHROOT, cap.SYS_PTRACE, cap.SYS_PACCT, cap.SYS_ADMIN, cap.SETUID)
				err = newcaps.SetFlag(cap.Effective, false, cap.CHOWN, cap.DAC_OVERRIDE, cap.DAC_READ_SEARCH, cap.FOWNER, cap.FSETID, cap.KILL, cap.SETGID, cap.SETUID, cap.SETPCAP, cap.LINUX_IMMUTABLE, cap.SYS_MODULE, cap.SYS_CHROOT, cap.SYS_PTRACE, cap.SYS_PACCT, cap.SYS_ADMIN, cap.SETUID)
				if err != nil {
					panic(err)
				}
			*/
			err = newcaps.SetFlag(cap.Effective, true, cap.NET_RAW, cap.NET_BIND_SERVICE)
			err = newcaps.SetFlag(cap.Inheritable, false, cap.NET_RAW, cap.NET_BIND_SERVICE)
			err = newcaps.SetFlag(cap.Permitted, true, cap.NET_RAW, cap.NET_BIND_SERVICE)
			if err != nil {
				panic(err)
			}
			newcaps.SetProc()
			if err != nil {
				panic(err)
			}
			curcaps := cap.GetProc()
			e, _ := curcaps.GetFlag(cap.Effective, cap.NET_RAW)
			i, _ := curcaps.GetFlag(cap.Inheritable, cap.NET_RAW)
			p, _ := curcaps.GetFlag(cap.Permitted, cap.NET_RAW)

			fmt.Printf("\nCHECK EIP=%v|%v|%v mode:%v\n", e, i, p, cap.GetMode())
			fmt.Printf("CAPS=%v | %s\n", curcaps, curcaps)
			fmt.Printf("NCAPS=%v | %s\n", newcaps, newcaps)

			/*
				err = cap.SetUID(0)
				if err != nil {
					panic(err)
				}
			*/
		}
		syscalls := []string{
			"access",
			"arch_prctl",
			"bind",
			"brk",
			"clone",
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
			"prctl",
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
		}

		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, syscalls...); err != nil {
			panic(err)
		}
		fmt.Printf("Installed seccomp policy")

		// //err := newcaps.SetFlag(cap.Permitted, true, cap.NET_RAW, cap.SETUID)
		// //err = newcaps.SetFlag(cap.Inheritable, true, cap.NET_RAW, cap.SETUID)
		// //err = newcaps.SetFlag(cap.Effective, true, cap.NET_RAW, cap.SETUID)
		// if err != nil {
		// 	panic(fmt.Sprintf("could not set caps: %s", err))
		// }
		// err = newcaps.SetProc()
		// if err != nil {
		// 	panic(fmt.Sprintf("could net set new process caps: %s", err))
		// }
		//err = syscall.Setuid(1000)

		//fmt.Printf("SET USER ID %s\n", newcaps)
	}
}
