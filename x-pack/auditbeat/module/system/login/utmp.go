// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package login

// #include <utmp.h>
// #include <stdlib.h>
import "C"

import (
	"encoding/binary"
	"net"
	"strconv"
	"time"
	"unsafe"

	"github.com/OneOfOne/xxhash"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

type RecordType int

const (
	Unknown RecordType = iota
	UserLogin
	UserLogout
)

var recordTypeToString = map[RecordType]string{
	Unknown:    "unknown",
	UserLogin:  "user_login",
	UserLogout: "user_logout",
}

// LoginRecord represents a login record.
type LoginRecord struct {
	utmp      Utmp
	Type      RecordType
	PID       int
	TTY       string
	UID       int
	Username  string
	Hostname  string
	IP        net.IP
	Timestamp time.Time
}

// Utmp contains data from the C utmp struct.
type Utmp struct {
	utType   int
	utPid    int
	utLine   string
	utUser   string
	utHost   string
	utTv     time.Time
	utAddrV6 [4]uint32
}

// String returns the string representation for a RecordType.
func (recordType RecordType) String() string {
	s, found := recordTypeToString[recordType]
	if found {
		return s
	} else {
		return ""
	}
}

// Hash creates a hash for LoginRecord.
func (login LoginRecord) Hash() uint64 {
	h := xxhash.New64()
	h.WriteString(strconv.Itoa(login.PID))
	h.WriteString(login.Timestamp.String())
	return h.Sum64()
}

func (login LoginRecord) toMapStr() common.MapStr {
	mapstr := common.MapStr{
		"type":      login.Type.String(),
		"pid":       login.PID,
		"tty":       login.TTY,
		"timestamp": login.Timestamp,
	}

	if login.Username != "" {
		mapstr.Put("user", common.MapStr{
			"name": login.Username,
		})
	}

	if login.Hostname != "" {
		mapstr.Put("hostname", login.Hostname)
	}

	if !login.IP.IsUnspecified() {
		mapstr.Put("ip", login.IP)
	}

	return mapstr
}

// ReadUtmpFile reads a UTMP formatted file (usually /var/log/wtmp).
func ReadUtmpFile(utmpFile string) ([]LoginRecord, error) {
	cs := C.CString(utmpFile)
	defer C.free(unsafe.Pointer(cs))

	success, err := C.utmpname(cs)
	if err != nil {
		return nil, errors.Wrap(err, "error selecting UTMP file")
	}
	if success != 0 {
		return nil, errors.New("selecting UTMP file failed")
	}

	C.setutent()
	defer C.endutent()

	var loginRecords []LoginRecord
	for {
		utmp, err := C.getutent()
		if err != nil {
			return nil, errors.Wrap(err, "error getting entry in UTMP file")
		}

		if utmp != nil {
			loginRecords = append(loginRecords, createLoginRecord(utmp))
		} else {
			break
		}
	}

	return loginRecords, nil
}

func createLoginRecord(utmpC *C.struct_utmp) LoginRecord {
	// See utmp(5) for the utmp struct fields.
	utmp := Utmp{
		utType:   int(utmpC.ut_type),
		utPid:    int(utmpC.ut_pid),
		utLine:   C.GoString(&utmpC.ut_line[0]),
		utUser:   C.GoString(&utmpC.ut_user[0]),
		utHost:   C.GoString(&utmpC.ut_host[0]),
		utTv:     time.Unix(int64(utmpC.ut_tv.tv_sec), int64(utmpC.ut_tv.tv_usec)*1000),
		utAddrV6: [4]uint32{uint32(utmpC.ut_addr_v6[0]), uint32(utmpC.ut_addr_v6[1]), uint32(utmpC.ut_addr_v6[2]), uint32(utmpC.ut_addr_v6[3])},
	}

	record := LoginRecord{
		utmp:      utmp,
		Timestamp: utmp.utTv,
		PID:       utmp.utPid,
		TTY:       utmp.utLine,
		UID:       -1,
	}

	switch utmp.utType {
	// See utmp(5) for C constants.
	case C.USER_PROCESS:
		record.Type = UserLogin
		record.Username = utmp.utUser
		record.IP = createIP(utmp.utAddrV6)
		record.Hostname = utmp.utHost
	case C.DEAD_PROCESS:
		record.Type = UserLogout
	default:
		record.Type = Unknown
	}

	return record
}

func createIP(utAddrV6 [4]uint32) net.IP {
	// See utmp(5) for the utmp struct fields.
	if utAddrV6[1] != 0 || utAddrV6[2] != 0 || utAddrV6[3] != 0 {
		// IPv6
		b := make([]byte, 16)
		binary.LittleEndian.PutUint32(b[:4], utAddrV6[0])
		binary.LittleEndian.PutUint32(b[4:8], utAddrV6[1])
		binary.LittleEndian.PutUint32(b[8:12], utAddrV6[2])
		binary.LittleEndian.PutUint32(b[12:], utAddrV6[3])
		return net.IP(b)
	} else {
		// IPv4
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, utAddrV6[0])
		return net.IP(b)
	}
}
