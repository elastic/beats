// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package types

//go:generate stringer -linecomment=true -type=Type,HookPoint,Field -output=gen_types_string.go

type Type uint64

const (
	ProcessFork Type = iota
	ProcessExec
	ProcessExit
	ProcessSetsid
)

type (
	Field uint32
)

const (
	Cwd Field = iota + 1
	Argv
	Env
	Filename
)

type PidInfo struct {
	StartTimeNs uint64
	Tid         uint32
	Tgid        uint32
	Vpid        uint32
	Ppid        uint32
	Pgid        uint32
	Sid         uint32
}

type CredInfo struct {
	Ruid         uint32
	Rgid         uint32
	Euid         uint32
	Egid         uint32
	Suid         uint32
	Sgid         uint32
	CapPermitted uint64
	CapEffective uint64
}

type TtyWinsize struct {
	Rows uint16
	Cols uint16
}

type TtyTermios struct {
	CIflag uint32
	COflag uint32
	CLflag uint32
	CCflag uint32
}

type TtyDev struct {
	Minor   uint16
	Major   uint16
	Winsize TtyWinsize
	Termios TtyTermios
}

type ProcessForkEvent struct {
	ParentPids PidInfo
	ChildPids  PidInfo
	Creds      CredInfo
}

type ProcessExecEvent struct {
	Pids  PidInfo
	Creds CredInfo
	CTty  TtyDev

	// varlen fields
	Cwd      string
	Argv     []string
	Env      map[string]string
	Filename string
}

type ProcessExitEvent struct {
	Pids     PidInfo
	ExitCode int32
}

type ProcessSetsidEvent struct {
	Pids PidInfo
}
