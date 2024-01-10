// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processdb

import (
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/types"
)

type DB interface {
	InsertFork(fork types.ProcessForkEvent) error
	InsertExec(exec types.ProcessExecEvent) error
	InsertSetsid(setsid types.ProcessSetsidEvent) error
	InsertExit(exit types.ProcessExitEvent) error
	GetProcess(pid uint32) (types.Process, error)
	ScrapeProcfs() []uint32
}

type TtyType int

const (
	TtyUnknown TtyType = iota
	Pts
	Tty
	TtyConsole
)

const (
	ptsMinMajor     = 136
	ptsMaxMajor     = 143
	ttyMajor        = 4
	consoleMaxMinor = 63
	ttyMaxMinor     = 255
)

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
