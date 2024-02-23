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

package security

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"

	"kernel.org/pub/linux/libs/security/libcap/cap"
)

func init() {
	// Here we set a bunch of linux specific security stuff.
	// In the context of a container, where users frequently run as root, we follow BEAT_SETUID_AS to setuid/gid
	// and add capabilities to make this actually run as a regular user. This also helps Node.js in synthetics, which
	// does not want to run as root. It's also just generally more secure.
	if localUserName := os.Getenv("BEAT_SETUID_AS"); localUserName != "" && syscall.Geteuid() == 0 {
		err := setNodeProcAttr(localUserName)
		if err != nil {
			panic(err)
		}
	}

	// Attempt to set capabilities before we setup seccomp rules
	// Note that we discard any errors because they are not actionable.
	// The beat should use `getcap` at a later point to examine available capabilities
	// rather than relying on errors from `setcap`
	_ = setCapabilities()
}

func setNodeProcAttr(localUserName string) error {

	localUser, err := user.Lookup(localUserName)
	if err != nil {
		return fmt.Errorf("could not lookup '%s': %w", localUser, err)
	}
	localUserUID, err := strconv.Atoi(localUser.Uid)
	if err != nil {
		return fmt.Errorf("could not parse UID '%s' as int: %w", localUser.Uid, err)
	}
	localUserGID, err := strconv.Atoi(localUser.Gid)
	if err != nil {
		return fmt.Errorf("could not parse GID '%s' as int: %w", localUser.Uid, err)
	}

	// We include the root group because the docker image contains many directories (data,logs)
	// that are owned by root:root with 0775 perms. The heartbeat user is in both groups
	// in the container, but we need to repeat that here.
	NodeChildProcCred = &syscall.Credential{
		Uid:         uint32(localUserUID),
		Gid:         uint32(localUserGID),
		Groups:      []uint32{0},
		NoSetGroups: false,
	}

	return os.Setenv("HOME", localUser.HomeDir)
}

func setCapabilities() error {
	newcaps := cap.GetProc()

	// Raise all permitted caps to effective
	err := newcaps.Fill(cap.Effective, cap.Permitted)
	if err != nil {
		return fmt.Errorf("error raising effective cap set: %w", err)
	}

	// Drop all inheritable caps to stop propagation to child proc
	err = newcaps.ClearFlag(cap.Inheritable)
	if err != nil {
		return fmt.Errorf("error clearing inheritable cap set: %w", err)
	}

	// Apply the new capabilities to the current process (incl. all threads)
	err = newcaps.SetProc()
	if err != nil {
		return fmt.Errorf("error setting new process capabilities via setcap: %w", err)
	}

	return nil
}
