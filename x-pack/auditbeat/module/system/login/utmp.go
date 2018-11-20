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
	"github.com/elastic/beats/libbeat/logp"
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
	Type      RecordType
	PID       int
	TTY       string
	Username  string
	Hostname  string
	IP        net.IP
	Timestamp time.Time
	utmp      *C.struct_utmp // Raw C structure
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

// UtmpDebugString returns a string containing
/*func (loginRecord *LoginRecord) UtmpDebugString() string {
	utmp := loginRecord.utmp

	utType := int(utmp.ut_type)
	utPid := int(utmp.ut_pid)
	utLine := C.GoString(&utmp.ut_line[0])
	utUser := C.GoString(&utmp.ut_user[0])
	utHost := C.GoString(&utmp.ut_host[0])
	utTvSec := int64(utmp.ut_tv.tv_sec)
	utTvUsec := int64(utmp.ut_tv.tv_usec)
	utAddrV6 := []uint32{uint32(utmp.ut_addr_v6[0]), uint32(utmp.ut_addr_v6[1]), uint32(utmp.ut_addr_v6[2]), uint32(utmp.ut_addr_v6[3])}

	return fmt.Sprintf("utmp: (ut_type=%d, ut_pid=%d, ut_line=%v, ut_user=%v, ut_host=%v, ut_tv.tv_sec=%d, ut_tv.tv_usec=%d, ut_addr_v6=%v)",
		utType, utPid, utLine, utUser, utHost, utTvSec, utTvUsec, utAddrV6)
}*/

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
		"ip":        login.IP,
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

	return mapstr
}

// ReadUtmpFile reads a UTMP formatted file (usually /var/log/wtmp).
func ReadUtmpFile(log *logp.Logger, utmpFile string) ([]LoginRecord, error) {
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
			loginRecords = append(loginRecords, createLoginRecord(log, utmp))
		} else {
			break
		}
	}

	return loginRecords, nil
}

func createLoginRecord(log *logp.Logger, utmp *C.struct_utmp) LoginRecord {
	utType := int(utmp.ut_type)
	utPid := int(utmp.ut_pid)
	utLine := C.GoString(&utmp.ut_line[0])
	utUser := C.GoString(&utmp.ut_user[0])
	utHost := C.GoString(&utmp.ut_host[0])
	utTvSec := int64(utmp.ut_tv.tv_sec)
	utTvUsec := int64(utmp.ut_tv.tv_usec)
	utAddrV6 := []uint32{uint32(utmp.ut_addr_v6[0]), uint32(utmp.ut_addr_v6[1]), uint32(utmp.ut_addr_v6[2]), uint32(utmp.ut_addr_v6[3])}
	log.Debugf("utmp: (ut_type=%d, ut_pid=%d, ut_line=%v, ut_user=%v, ut_host=%v, ut_tv.tv_sec=%d, ut_tv.tv_usec=%d, ut_addr_v6=%v)",
		utType, utPid, utLine, utUser, utHost, utTvSec, utTvUsec, utAddrV6)

	record := LoginRecord{
		PID:       utPid,
		TTY:       utLine,
		Hostname:  utHost,
		Timestamp: time.Unix(utTvSec, utTvUsec*1000),
	}

	// See utmp(5) for the utmp struct fields.
	if uint32(utmp.ut_addr_v6[1]) != 0 || uint32(utmp.ut_addr_v6[2]) != 0 || uint32(utmp.ut_addr_v6[3]) != 0 {
		// IPv6
		b := make([]byte, 16)
		binary.LittleEndian.PutUint32(b[:4], uint32(utmp.ut_addr_v6[0]))
		binary.LittleEndian.PutUint32(b[4:8], uint32(utmp.ut_addr_v6[1]))
		binary.LittleEndian.PutUint32(b[8:12], uint32(utmp.ut_addr_v6[2]))
		binary.LittleEndian.PutUint32(b[12:], uint32(utmp.ut_addr_v6[3]))
		record.IP = net.IP(b)
	} else {
		// IPv4
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(utmp.ut_addr_v6[0]))
		record.IP = net.IP(b)
	}

	switch utType {
	// See utmp(5) for C constants.
	case C.USER_PROCESS:
		record.Type = UserLogin
		record.Username = utUser
	case C.DEAD_PROCESS:
		record.Type = UserLogout
	default:
		record.Type = Unknown
	}

	return record
}
