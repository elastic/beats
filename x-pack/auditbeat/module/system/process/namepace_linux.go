// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package process

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// isNsSharedWith returns whether the process with the given pid shares the
// namespace ns with the current process.
func isNsSharedWith(pid int, ns string) (yes bool, err error) {
	self, err := selfNsIno(ns)
	if err != nil {
		return false, err
	}
	other, err := nsIno(pid, ns)
	if err != nil {
		return false, err
	}
	return self == other, nil
}

// selfNsIno returns the inode number for the namespace ns for this process.
func selfNsIno(ns string) (ino uint64, err error) {
	fi, err := os.Stat(fmt.Sprintf("/proc/self/ns/%s", ns))
	if err != nil {
		return 0, err
	}
	sysInfo, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("not a stat_t")
	}
	return sysInfo.Ino, nil
}

// nsIno returns the inode number for the namespace ns for the process with
// the given pid.
func nsIno(pid int, ns string) (ino uint64, err error) {
	fi, err := os.Stat(fmt.Sprintf("/proc/%d/ns/%s", pid, ns))
	if err != nil {
		return 0, err
	}
	sysInfo, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("not a stat_t")
	}
	return sysInfo.Ino, nil
}
