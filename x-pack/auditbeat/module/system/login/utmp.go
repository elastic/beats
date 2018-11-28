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
	"reflect"
	"time"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
)

type FileRecord struct {
	Inode           uint64
	LastCtime       time.Time
	LastLoginRecord LoginRecord
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

func newUtmp(utmpC *C.struct_utmp) Utmp {
	// See utmp(5) for the utmp struct fields.
	return Utmp{
		utType:   int(utmpC.ut_type),
		utPid:    int(utmpC.ut_pid),
		utLine:   C.GoString(&utmpC.ut_line[0]),
		utUser:   C.GoString(&utmpC.ut_user[0]),
		utHost:   C.GoString(&utmpC.ut_host[0]),
		utTv:     time.Unix(int64(utmpC.ut_tv.tv_sec), int64(utmpC.ut_tv.tv_usec)*1000),
		utAddrV6: [4]uint32{uint32(utmpC.ut_addr_v6[0]), uint32(utmpC.ut_addr_v6[1]), uint32(utmpC.ut_addr_v6[2]), uint32(utmpC.ut_addr_v6[3])},
	}
}

type UtmpFileReader struct {
	log *logp.Logger
}

// ReadAfter reads a UTMP formatted file (usually /var/log/wtmp*)
// and returns the records after the provided last known record.
// If record is nil, it returns all records in the file.
func (r *UtmpFileReader) ReadAfter(utmpFile string, lastKnownRecord *LoginRecord) ([]LoginRecord, error) {
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

	reachedNewRecords := (lastKnownRecord == nil)
	var loginRecords []LoginRecord
	for {
		utmpC, err := C.getutent()
		if err != nil {
			return nil, errors.Wrap(err, "error getting entry in UTMP file")
		}

		if utmpC != nil {
			utmp := newUtmp(utmpC)

			if reachedNewRecords {
				r.log.Debugf("utmp: (ut_type=%d, ut_pid=%d, ut_line=%v, ut_user=%v, ut_host=%v, ut_tv.tv_sec=%v, ut_addr_v6=%v)",
					utmp.utType, utmp.utPid, utmp.utLine, utmp.utUser, utmp.utHost, utmp.utTv, utmp.utAddrV6)

				loginRecords = append(loginRecords, createLoginRecord(utmp))
			}

			if lastKnownRecord != nil && reflect.DeepEqual(utmp, lastKnownRecord.Utmp) {
				reachedNewRecords = true
			}
		} else {
			break
		}
	}

	return loginRecords, nil
}

func createLoginRecord(utmp Utmp) LoginRecord {
	record := LoginRecord{
		Utmp:      utmp,
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
