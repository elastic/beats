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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
)

type FileRecord struct {
	Inode    uint64
	Size     int64
	LastUtmp Utmp
}

// Utmp contains data from the C utmp struct.
type Utmp struct {
	UtType   int
	UtPid    int
	UtLine   string
	UtUser   string
	UtHost   string
	UtTv     time.Time
	UtAddrV6 [4]uint32
}

func newUtmp(utmpC *C.struct_utmp) Utmp {
	// See utmp(5) for the utmp struct fields.
	return Utmp{
		UtType:   int(utmpC.ut_type),
		UtPid:    int(utmpC.ut_pid),
		UtLine:   C.GoString(&utmpC.ut_line[0]),
		UtUser:   C.GoString(&utmpC.ut_user[0]),
		UtHost:   C.GoString(&utmpC.ut_host[0]),
		UtTv:     time.Unix(int64(utmpC.ut_tv.tv_sec), int64(utmpC.ut_tv.tv_usec)*1000),
		UtAddrV6: [4]uint32{uint32(utmpC.ut_addr_v6[0]), uint32(utmpC.ut_addr_v6[1]), uint32(utmpC.ut_addr_v6[2]), uint32(utmpC.ut_addr_v6[3])},
	}
}

type UtmpFileReader struct {
	log         *logp.Logger
	filePattern string
	fileRecords map[uint64]FileRecord
}

// ReadNew returns any new UTMP entries in any of the configured UTMP formatted files (usually /var/log/wtmp).
func (r *UtmpFileReader) ReadNew() ([]LoginRecord, error) {
	fileInfos, err := r.fileInfos()
	defer r.deleteOldRecords(fileInfos)

	var loginRecords []LoginRecord
	for path, fileInfo := range fileInfos {
		inode := fileInfo.Sys().(*syscall.Stat_t).Ino

		fileRecord, isKnownFile := r.fileRecords[inode]
		var oldSize int64 = 0
		if isKnownFile {
			oldSize = fileRecord.Size
		}
		newSize := fileInfo.Size()
		if !isKnownFile || newSize != oldSize {
			r.log.Debugf("Reading file %v (inode=%v, oldSize=%v, newSize=%v)", path, inode, oldSize, newSize)

			var utmpRecords []Utmp

			// Once we start reading a file, we update the file record even if something fails -
			// otherwise we will just keep trying to re-read very frequently forever.
			defer r.updateFileRecord(inode, newSize, &utmpRecords)

			if isKnownFile {
				utmpRecords, err = r.readAfter(path, &fileRecord.LastUtmp)
			} else {
				utmpRecords, err = r.readAfter(path, nil)
			}

			if err != nil {
				return nil, errors.Wrapf(err, "error reading file %v", path)
			} else if len(utmpRecords) == 0 {
				return nil, fmt.Errorf("unexpectedly, there are no new records in file %v", path)
			} else {
				for _, utmp := range utmpRecords {
					loginRecords = append(loginRecords, newLoginRecord(utmp))
				}
			}
		}
	}

	return loginRecords, nil
}

// deleteOldRecords clean up old file records where the inode no longer exists.
func (r *UtmpFileReader) deleteOldRecords(fileInfos map[string]os.FileInfo) {
	for savedInode, _ := range r.fileRecords {
		found := false
		for _, fileInfo := range fileInfos {
			inode := fileInfo.Sys().(*syscall.Stat_t).Ino
			if inode == savedInode {
				found = true
				break
			}
		}

		if !found {
			r.log.Debugf("Deleting file record for old inode %d", savedInode)
			delete(r.fileRecords, savedInode)
		}
	}
}

func (r *UtmpFileReader) fileInfos() (map[string]os.FileInfo, error) {
	paths, err := filepath.Glob(r.filePattern)
	if err != nil {
		return nil, errors.Wrap(err, "failed to expand file pattern")
	}

	// Sort paths in reverse order (oldest/most-rotated file first)
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))

	fileInfos := make(map[string]os.FileInfo, len(r.fileRecords))
	for _, path := range paths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				// Skip - file might have been rotated out
				r.log.Debugf("File %v does not exist anymore.", path)
				continue
			} else {
				return nil, errors.Wrapf(err, "unexpected error when reading file %v", path)
			}
		} else if fileInfo.Sys() == nil {
			return nil, fmt.Errorf("empty stat result for file %v", path)
		}

		fileInfos[path] = fileInfo
	}

	return fileInfos, nil
}

func (r *UtmpFileReader) updateFileRecord(inode uint64, size int64, utmpRecords *[]Utmp) {
	newFileRecord := FileRecord{
		Inode: inode,
		Size:  size,
	}

	if len(*utmpRecords) > 0 {
		newFileRecord.LastUtmp = (*utmpRecords)[len(*utmpRecords)-1]
	} else {
		oldFileRecord, found := r.fileRecords[inode]
		if found {
			newFileRecord.LastUtmp = oldFileRecord.LastUtmp
		}
	}

	r.fileRecords[inode] = newFileRecord
}

// ReadAfter reads a UTMP formatted file (usually /var/log/wtmp*)
// and returns the records after the provided last known record.
// If record is nil, it returns all records in the file.
func (r *UtmpFileReader) readAfter(utmpFile string, lastKnownRecord *Utmp) ([]Utmp, error) {
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
	var utmpRecords []Utmp
	for {
		utmpC, err := C.getutent()
		if err != nil {
			return nil, errors.Wrap(err, "error getting entry in UTMP file")
		}

		if utmpC != nil {
			utmp := newUtmp(utmpC)

			if reachedNewRecords {
				r.log.Debugf("utmp: (ut_type=%d, ut_pid=%d, ut_line=%v, ut_user=%v, ut_host=%v, ut_tv.tv_sec=%v, ut_addr_v6=%v)",
					utmp.UtType, utmp.UtPid, utmp.UtLine, utmp.UtUser, utmp.UtHost, utmp.UtTv, utmp.UtAddrV6)

				utmpRecords = append(utmpRecords, utmp)
			}

			if lastKnownRecord != nil && reflect.DeepEqual(utmp, *lastKnownRecord) {
				reachedNewRecords = true
			}
		} else {
			// Eventually, we have read all UTMP records in the file.
			break
		}
	}

	return utmpRecords, nil
}

func newLoginRecord(utmp Utmp) LoginRecord {
	record := LoginRecord{
		Utmp:      utmp,
		Timestamp: utmp.UtTv,
		PID:       utmp.UtPid,
		TTY:       utmp.UtLine,
		UID:       -1,
	}

	switch utmp.UtType {
	// See utmp(5) for C constants.
	case C.USER_PROCESS:
		record.Type = UserLogin
		record.Username = utmp.UtUser
		record.IP = createIP(utmp.UtAddrV6)
		record.Hostname = utmp.UtHost
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
