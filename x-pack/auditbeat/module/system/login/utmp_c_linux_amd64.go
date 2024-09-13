// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

// Pure Go reader for UTMP formatted files.
// See utmp(5) and getutent(3) for the C structs and functions this is
// replacing.

package login

// utmpC is a Go representation of the C utmp struct that the UTMP files consist of.
type utmpC struct {
	Type UtType

	// Alignment
	_ [2]byte

	Pid      int32
	Device   [UT_LINESIZE]byte
	Terminal [4]byte
	Username [UT_NAMESIZE]byte
	Hostname [UT_HOSTSIZE]byte

	ExitStatusTermination int16
	ExitStatusExit        int16

	SessionID int32

	TimeSeconds      int32
	TimeMicroseconds int32

	IP [4]int32

	Unused [20]byte
}
