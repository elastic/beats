// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
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
	"unsafe"
)

var byteOrder = getByteOrder()

func getByteOrder() binary.ByteOrder {
	var b [2]byte
	*((*uint16)(unsafe.Pointer(&b[0]))) = 1
	if b[0] == 1 {
		return binary.LittleEndian
	}
	return binary.BigEndian
}

// UtType represents the ut_type field. See utmp(5).
type UtType int16

// Possible values for UtType.
const (
	EMPTY         UtType = 0
	RUN_LVL       UtType = 1
	BOOT_TIME     UtType = 2
	NEW_TIME      UtType = 3
	OLD_TIME      UtType = 4
	INIT_PROCESS  UtType = 5
	LOGIN_PROCESS UtType = 6
	USER_PROCESS  UtType = 7
	DEAD_PROCESS  UtType = 8
	ACCOUNTING    UtType = 9

	UT_LINESIZE = 32
	UT_NAMESIZE = 32
	UT_HOSTSIZE = 256
)

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

// Utmp contains a Go version of UtmpC.
type Utmp struct {
	UtType   UtType
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
		UtType:   utmp.Type,
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
	if n == -1 {
		n = len(b)
	}
	return string(b[:n])
}

// ReadNextUtmp reads the next UTMP entry in a reader pointing to UTMP formatted data.
func ReadNextUtmp(r io.Reader) (*Utmp, error) {
	utmpC := new(utmpC)

	err := binary.Read(r, byteOrder, utmpC)
	if err != nil {
		return nil, err
	}

	return newUtmp(utmpC), nil
}
