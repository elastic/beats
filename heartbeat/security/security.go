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

//go:build linux
// +build linux

package security

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"syscall"

	"github.com/elastic/go-sysinfo"

	"kernel.org/pub/linux/libs/security/libcap/cap"

	"github.com/elastic/beats/v7/libbeat/common/seccomp"
	seccomp_default "github.com/elastic/go-seccomp-bpf"
)

func init() {
	// Here we set a bunch of linux specific security stuff.
	// In the context of a container, where users frequently run as root, we follow BEAT_SETUID_AS to setuid/gid
	// and add capabilities to make this actually run as a regular user. This also helps Node.js in synthetics, which
	// does not want to run as root. It's also just generally more secure.
	sysInfo, err := sysinfo.Host()
	isContainer := false
	if err == nil && sysInfo.Info().Containerized != nil {
		isContainer = *sysInfo.Info().Containerized
	}

	if localUserName := os.Getenv("BEAT_SETUID_AS"); isContainer && localUserName != "" && syscall.Geteuid() == 0 {
		err := changeUser(localUserName)
		if err != nil {
			panic(err)
		}
	}

	// Attempt to set capabilities before we setup seccomp rules
	// Note that we discard any errors because they are not actionable.
	// The beat should use `getcap` at a later point to examine available capabilities
	// rather than relying on errors from `setcap`
	_ = setCapabilities()

	err = setSeccompRules()
	if err != nil {
		panic(err)
	}
}

func changeUser(localUserName string) error {
	localUser, err := user.Lookup(localUserName)
	if err != nil {
		return fmt.Errorf("could not lookup '%s': %w", localUser, err)
	}
	localUserUid, err := strconv.Atoi(localUser.Uid)
	if err != nil {
		return fmt.Errorf("could not parse UID '%s' as int: %w", localUser.Uid, err)
	}
	localUserGid, err := strconv.Atoi(localUser.Gid)
	if err != nil {
		return fmt.Errorf("could not parse GID '%s' as int: %w", localUser.Uid, err)
	}
	// We include the root group because the docker image contains many directories (data,logs)
	// that are owned by root:root with 0775 perms. The heartbeat user is in both groups
	// in the container, but we need to repeat that here.
	err = syscall.Setgroups([]int{localUserGid, 0})
	if err != nil {
		return fmt.Errorf("could not set groups: %w", err)
	}

	// Set the main group as localUserUid so new files created are owned by the user's group
	err = syscall.Setgid(localUserGid)
	if err != nil {
		return fmt.Errorf("could not set gid to %d: %w", localUserGid, err)
	}

	// Note this is not the regular SetUID! Look at the 'cap' package docs for it, it preserves
	// capabilities post-SetUID, which we use to lock things down immediately
	err = cap.SetUID(localUserUid)
	if err != nil {
		return fmt.Errorf("could not setuid to %d: %w", localUserUid, err)
	}

	// This may not be necessary, but is good hygiene, we do some shelling out to node/npm etc.
	// and $HOME should reflect the user's preferences
	return os.Setenv("HOME", localUser.HomeDir)
}

func setCapabilities() error {
	// Start with an empty capability set
	newcaps := cap.NewSet()
	// Both permitted and effective are required! Permitted makes the permmission
	// possible to get, effective makes it 'active'
	err := newcaps.SetFlag(cap.Permitted, true, cap.NET_RAW)
	if err != nil {
		return fmt.Errorf("error setting permitted setcap: %w", err)
	}
	err = newcaps.SetFlag(cap.Effective, true, cap.NET_RAW)
	if err != nil {
		return fmt.Errorf("error setting effective setcap: %w", err)
	}

	// We do not want these capabilities to be inherited by subprocesses
	err = newcaps.SetFlag(cap.Inheritable, false, cap.NET_RAW)
	if err != nil {
		return fmt.Errorf("error setting inheritable setcap: %w", err)
	}

	// Apply the new capabilities to the current process (incl. all threads)
	err = newcaps.SetProc()
	if err != nil {
		return fmt.Errorf("error setting new process capabilities via setcap: %w", err)
	}

	return nil
}

func setSeccompRules() error {
	// We require a number of syscalls to run. This list was generated with
	// mage build && env ELASTIC_SYNTHETICS_CAPABLE=true strace -f --output=syscalls  ./heartbeat --path.config sample-synthetics-config/ -e
	// then grepping for 'EPERM' in the 'syscalls' file.
	switch runtime.GOARCH {
	case "amd64", "386":
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
		return seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, syscalls...)

	case "arm64", "aarch64":
		// Register default arm64/aarch64 policy
		defaultPolicy := &seccomp_default.Policy{
			DefaultAction: seccomp_default.ActionAllow,
			Syscalls: []seccomp_default.SyscallGroup{
				{
					Action: seccomp_default.ActionErrno,
					Names: []string{
						"execveat",
					},
				},
			},
		}
		seccomp.MustRegisterPolicy(defaultPolicy)

	}

	return nil
}
