// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

// Pure Go reader for UTMP formatted files.
// See utmp(5) and getutent(3) for the C structs and functions this is
// replacing.

package login

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
)

// Possible values for the ut_type field. See utmp(5).
const (
	EMPTY         = 0
	RUN_LVL       = 1
	BOOT_TIME     = 2
	NEW_TIME      = 3
	OLD_TIME      = 4
	INIT_PROCESS  = 5
	LOGIN_PROCESS = 6
	USER_PROCESS  = 7
	DEAD_PROCESS  = 8
	ACCOUNTING    = 9

	UT_LINESIZE = 32
	UT_NAMESIZE = 32
	UT_HOSTSIZE = 256
)

// utmpC is a Go representation of the C utmp struct that the UTMP files consist of.
type utmpC struct {
	Type int16

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

// Utmp contains a Go version of UtmpC.
type Utmp struct {
	UtType   int
	UtPid    int
	UtLine   string
	UtUser   string
	UtHost   string
	UtTv     time.Time
	UtAddrV6 [4]uint32
}

// newUtmp creates a Utmp out of a utmpC.
func newUtmp(utmp *utmpC) *Utmp {
	// See utmp(5) for the utmp struct fields.
	return &Utmp{
		UtType:   int(utmp.Type),
		UtPid:    int(utmp.Pid),
		UtLine:   byteToString(utmp.Device[:]),
		UtUser:   byteToString(utmp.Username[:]),
		UtHost:   byteToString(utmp.Hostname[:]),
		UtTv:     time.Unix(int64(utmp.TimeSeconds), int64(utmp.TimeMicroseconds)*1000),
		UtAddrV6: [4]uint32{uint32(utmp.IP[0]), uint32(utmp.IP[1]), uint32(utmp.IP[2]), uint32(utmp.IP[3])},
	}
}

// byteToString converts a NULL terminated char array to a Go string.
func byteToString(b []byte) string {
	n := bytes.IndexByte(b, 0)
	return string(b[:n])
}

// ReadNextUtmp reads the next UTMP entry in a reader pointing to UTMP formatted data.
func ReadNextUtmp(r io.Reader) (*Utmp, error) {
	utmpC := new(utmpC)

	err := binary.Read(r, binary.LittleEndian, utmpC)
	if err != nil {
		return nil, err
	}

	return newUtmp(utmpC), nil
}
