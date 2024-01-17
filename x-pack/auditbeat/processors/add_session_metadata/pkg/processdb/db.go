// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processdb

import (
	"strings"

	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/types"
)

type DB interface {
	InsertFork(fork types.ProcessForkEvent) error
	InsertExec(exec types.ProcessExecEvent) error
	InsertSetsid(setsid types.ProcessSetsidEvent) error
	InsertExit(exit types.ProcessExitEvent) error
	GetProcess(pid uint32) (types.Process, error)
	GetEntryType(pid uint32) (EntryType, error)
	ScrapeProcfs() []uint32
}

type TtyType int

const (
	TtyUnknown TtyType = iota
	Pts
	Tty
	TtyConsole
)

type EntryType string

const (
	Init         EntryType = "init"
	Sshd         EntryType = "sshd"
	Ssm          EntryType = "ssm"
	Container    EntryType = "container"
	Terminal     EntryType = "terminal"
	EntryConsole EntryType = "console"
	EntryUnknown EntryType = "unknown"
)

var containerRuntimes = [...]string{
	"containerd-shim",
	"runc",
	"conmon",
}

// "filtered" executables are executables that relate to internal
// implementation details of entry mechanisms. The set of circumstances under
// which they can become an entry leader are reduced compared to other binaries
// (see implementation and unit tests).
var filteredExecutables = [...]string{
	"runc",
	"containerd-shim",
	"calico-node",
	"check-status",
	"conmon",
}

const (
	ptsMinMajor     = 136
	ptsMaxMajor     = 143
	ttyMajor        = 4
	consoleMaxMinor = 63
	ttyMaxMinor     = 255
)

func stringStartsWithEntryInList(str string, list []string) bool {
	for _, entry := range list {
		if strings.HasPrefix(str, entry) {
			return true
		}
	}

	return false
}

func isContainerRuntime(executable string) bool {
	return slices.ContainsFunc(containerRuntimes[:], func(s string) bool {
		return strings.HasPrefix(executable, s)
	})
}

func isFilteredExecutable(executable string) bool {
	return stringStartsWithEntryInList(executable, filteredExecutables[:])
}

func getTtyType(major uint16, minor uint16) TtyType {
	if major >= ptsMinMajor && major <= ptsMaxMajor {
		return Pts
	}

	if ttyMajor == major {
		if minor <= consoleMaxMinor {
			return TtyConsole
		} else if minor > consoleMaxMinor && minor <= ttyMaxMinor {
			return Tty
		}
	}

	return TtyUnknown
}
