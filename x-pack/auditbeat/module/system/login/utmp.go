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

// See utmp(5).
var utTypes = map[int]string{
	0: "EMPTY",
	1: "RUN_LVL",   // keep - shutdown
	2: "BOOT_TIME", // keep - reboot
	3: "NEW_TIME",  // investigate
	4: "OLD_TIME",  // investigate
	5: "INIT_PROCESS",
	6: "LOGIN_PROCESS",
	7: "USER_PROCESS", // keep - user login
	8: "DEAD_PROCESS", // keep - user logout
	9: "ACCOUNTING",
}

// LoginRecord represents a login record.
type LoginRecord struct {
	RecordType string
	PID        int
	TTY        string
	Username   string
	Hostname   string
	IP         net.IP
	Timestamp  time.Time
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
		"record_type": login.RecordType,
		"pid":         login.PID,
		"tty":         login.TTY,
		"ip":          login.IP,
		"timestamp":   login.Timestamp,
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
			recordType, found := utTypes[int(utmp.ut_type)]
			if !found {
				recordType = strconv.Itoa(int(utmp.ut_type))
			}

			var ip net.IP
			if uint32(utmp.ut_addr_v6[1]) != 0 || uint32(utmp.ut_addr_v6[2]) != 0 || uint32(utmp.ut_addr_v6[3]) != 0 {
				// IPv6
				b := make([]byte, 16)
				binary.LittleEndian.PutUint32(b[:4], uint32(utmp.ut_addr_v6[0]))
				binary.LittleEndian.PutUint32(b[4:8], uint32(utmp.ut_addr_v6[1]))
				binary.LittleEndian.PutUint32(b[8:12], uint32(utmp.ut_addr_v6[2]))
				binary.LittleEndian.PutUint32(b[12:], uint32(utmp.ut_addr_v6[3]))
				ip = net.IP(b)
			} else {
				// IPv4
				b := make([]byte, 4)
				binary.LittleEndian.PutUint32(b, uint32(utmp.ut_addr_v6[0]))
				ip = net.IP(b)
			}

			// See utmp(5) for the utmp struct fields.
			loginRecords = append(loginRecords, LoginRecord{
				RecordType: recordType,
				PID:        int(utmp.ut_pid),
				TTY:        C.GoString(&utmp.ut_line[0]),
				Username:   C.GoString(&utmp.ut_user[0]),
				Hostname:   C.GoString(&utmp.ut_host[0]),
				IP:         ip,
				Timestamp:  time.Unix(int64(utmp.ut_tv.tv_sec), int64(utmp.ut_tv.tv_usec*1000)),
			})
		} else {
			break
		}
	}

	return loginRecords, nil
}
