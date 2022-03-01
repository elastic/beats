// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package login

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"

	"github.com/elastic/beats/v7/auditbeat/datastore"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	bucketKeyFileRecords   = "file_records"
	bucketKeyLoginSessions = "login_sessions"
)

// Inode represents a file's inode on Linux.
type Inode uint64

// UtmpType represents the type of a UTMP file and records.
// Two types are possible: wtmp (records from the "good" file, i.e. /var/log/wtmp)
// and btmp (failed logins from /var/log/btmp).
type UtmpType uint8

const (
	// Wtmp is the "normal" wtmp file that includes successful logins, logouts,
	// and system boots.
	Wtmp UtmpType = iota
	// Btmp contains bad logins only.
	Btmp
)

// UtmpFile represents a UTMP file at a point in time.
type UtmpFile struct {
	Inode  Inode
	Path   string
	Size   int64
	Offset int64
	Type   UtmpType
}

// UtmpFileReader can read a UTMP formatted file (usually /var/log/wtmp).
type UtmpFileReader struct {
	log            *logp.Logger
	bucket         datastore.Bucket
	config         config
	savedUtmpFiles map[Inode]UtmpFile
	loginSessions  map[string]LoginRecord
}

// NewUtmpFileReader creates and initializes a new UTMP file reader.
func NewUtmpFileReader(log *logp.Logger, bucket datastore.Bucket, config config) (*UtmpFileReader, error) {
	r := &UtmpFileReader{
		log:            log,
		bucket:         bucket,
		config:         config,
		savedUtmpFiles: make(map[Inode]UtmpFile),
		loginSessions:  make(map[string]LoginRecord),
	}

	// Load state (file records, tty mapping) from disk
	err := r.restoreStateFromDisk()
	if err != nil {
		return nil, fmt.Errorf("failed to restore state from disk: %w", err)
	}

	return r, nil
}

// Close performs any cleanup tasks when the UTMP reader is done.
func (r *UtmpFileReader) Close() error {
	if r.bucket != nil {
		return r.bucket.Close()
	}
	return nil
}

// ReadNew returns any new UTMP entries in any files matching the configured pattern.
func (r *UtmpFileReader) ReadNew() (<-chan LoginRecord, <-chan error) {
	loginRecordC := make(chan LoginRecord)
	errorC := make(chan error)

	go func() {
		defer logp.Recover("A panic occurred while collecting login information")
		defer close(loginRecordC)
		defer close(errorC)

		wtmpFiles, err := r.findFiles(r.config.WtmpFilePattern, Wtmp)
		if err != nil {
			errorC <- fmt.Errorf("failed to expand file pattern: %w", err)
			return
		}

		btmpFiles, err := r.findFiles(r.config.BtmpFilePattern, Btmp)
		if err != nil {
			errorC <- fmt.Errorf("failed to expand file pattern: %w", err)
			return
		}

		utmpFiles := append(wtmpFiles, btmpFiles...)
		defer r.deleteOldUtmpFiles(&utmpFiles)

		for _, utmpFile := range utmpFiles {
			r.readNewInFile(loginRecordC, errorC, utmpFile)
		}
	}()

	return loginRecordC, errorC
}

func (r *UtmpFileReader) findFiles(filePattern string, utmpType UtmpType) ([]UtmpFile, error) {
	paths, err := filepath.Glob(filePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to expand file pattern %v: %w", filePattern, err)
	}

	// Sort paths in reverse order (oldest/most-rotated file first)
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))

	var utmpFiles []UtmpFile
	for _, path := range paths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				// Skip - file might have been rotated out
				r.log.Debugf("File %v does not exist anymore.", path)
				continue
			} else {
				return nil, fmt.Errorf("unexpected error when looking up file %v: %w", path, err)
			}
		}

		utmpFiles = append(utmpFiles, UtmpFile{
			Inode: Inode(fileInfo.Sys().(*syscall.Stat_t).Ino),
			Path:  path,
			Size:  fileInfo.Size(),
			Type:  utmpType,
		})
	}

	return utmpFiles, nil
}

// deleteOldUtmpFiles cleans up old UTMP file records where the inode no longer exists.
func (r *UtmpFileReader) deleteOldUtmpFiles(existingFiles *[]UtmpFile) {
	existingInodes := make(map[Inode]struct{})
	for _, utmpFile := range *existingFiles {
		existingInodes[utmpFile.Inode] = struct{}{}
	}

	for savedInode := range r.savedUtmpFiles {
		if _, exists := existingInodes[savedInode]; !exists {
			r.log.Debugf("Deleting file record for old inode %d.", savedInode)
			delete(r.savedUtmpFiles, savedInode)
		}
	}
}

// readNewInFile reads a UTMP formatted file and emits the records after the last known record.
func (r *UtmpFileReader) readNewInFile(loginRecordC chan<- LoginRecord, errorC chan<- error, utmpFile UtmpFile) {
	savedUtmpFile, isKnownFile := r.savedUtmpFiles[utmpFile.Inode]
	if !isKnownFile {
		r.log.Debugf("Found new file: %v (utmpFile=%+v)", utmpFile.Path, utmpFile)
	}
	utmpFile.Offset = savedUtmpFile.Offset

	size := utmpFile.Size
	oldSize := savedUtmpFile.Size
	if size < oldSize || utmpFile.Offset > size {
		// UTMP files are append-only and so this is weird. It might be a sign of
		// a highly unlikely inode reuse - or of something more nefarious.
		// Setting isKnownFile to false so we read the whole file from the beginning.
		isKnownFile = false

		r.log.Warnf("saved size or offset illogical (new=%+v, saved=%+v) - reading whole file.",
			utmpFile, savedUtmpFile)
	}

	if !isKnownFile && size == 0 {
		// Empty new file - save but don't read.
		err := r.updateSavedUtmpFile(utmpFile, nil)
		if err != nil {
			errorC <- fmt.Errorf("error updating file record for file %v: %w", utmpFile.Path, err)
		}
		return
	}

	if !isKnownFile || size != oldSize {
		r.log.Debugf("Reading file %v (utmpFile=%+v)", utmpFile.Path, utmpFile)

		f, err := os.Open(utmpFile.Path)
		if err != nil {
			errorC <- fmt.Errorf("error opening file %v: %w", utmpFile.Path, err)
			return
		}
		defer func() {
			// Once we start reading a file, we update the file record even if something fails -
			// otherwise we will just keep trying to re-read very frequently forever.
			err := r.updateSavedUtmpFile(utmpFile, f)
			if err != nil {
				errorC <- fmt.Errorf("error updating file record for file %v: %w", utmpFile.Path, err)
			}

			f.Close()
		}()

		// This will be the usual case, but we do not want to seek with the stored offset
		// if the saved size is smaller than the current one.
		if size >= oldSize && utmpFile.Offset <= size {
			_, err = f.Seek(utmpFile.Offset, 0)
			if err != nil {
				errorC <- fmt.Errorf("error setting offset %d for file %v: %w", utmpFile.Offset, utmpFile.Path, err)
			}
		}

		// If the saved size is smaller than the current one, or the previous Seek failed,
		// we retry one more time, this time resetting to the beginning of the file.
		if size < oldSize || utmpFile.Offset > size || err != nil {
			_, err = f.Seek(0, 0)
			if err != nil {
				errorC <- fmt.Errorf("error setting offset 0 for file %v: %w", utmpFile.Path, err)

				// Even that did not work, so return.
				return
			}
		}

		for {
			utmp, err := ReadNextUtmp(f)
			if err != nil && err != io.EOF {
				errorC <- fmt.Errorf("error reading entry in UTMP file %v: %w", utmpFile.Path, err)
				return
			}

			if utmp != nil {
				r.log.Debugf("utmp: (ut_type=%d, ut_pid=%d, ut_line=%v, ut_user=%v, ut_host=%v, ut_tv.tv_sec=%v, ut_addr_v6=%v)",
					utmp.UtType, utmp.UtPid, utmp.UtLine, utmp.UtUser, utmp.UtHost, utmp.UtTv, utmp.UtAddrV6)

				var loginRecord *LoginRecord
				switch utmpFile.Type {
				case Wtmp:
					loginRecord = r.processGoodLoginRecord(utmp)
				case Btmp:
					loginRecord, err = r.processBadLoginRecord(utmp)
					if err != nil {
						errorC <- err
					}
				}

				if loginRecord != nil {
					loginRecord.Origin = utmpFile.Path
					loginRecordC <- *loginRecord
				}
			} else {
				// Eventually, we have read all UTMP records in the file.
				break
			}
		}
	}
}

func (r *UtmpFileReader) updateSavedUtmpFile(utmpFile UtmpFile, f *os.File) error {
	if f != nil {
		offset, err := f.Seek(0, 1)
		if err != nil {
			return fmt.Errorf("error calling Seek: %w", err)
		}
		utmpFile.Offset = offset
	}

	r.log.Debugf("Saving UTMP file record (%+v)", utmpFile)

	r.savedUtmpFiles[utmpFile.Inode] = utmpFile

	return nil
}

// processBadLoginRecord takes a UTMP login record from the "bad" login file (/var/log/btmp)
// and returns a LoginRecord for it.
func (r *UtmpFileReader) processBadLoginRecord(utmp *Utmp) (*LoginRecord, error) {
	record := LoginRecord{
		Utmp:      utmp,
		Timestamp: utmp.UtTv,
		TTY:       utmp.UtLine,
		UID:       -1,
		PID:       -1,
	}

	switch utmp.UtType {
	// See utmp(5) for C constants.
	case LOGIN_PROCESS, USER_PROCESS:
		record.Type = userLoginFailedRecord

		record.Username = utmp.UtUser
		record.UID = lookupUsername(record.Username)
		record.PID = utmp.UtPid
		record.IP = newIP(utmp.UtAddrV6)
		record.Hostname = utmp.UtHost
	default:
		// This should not happen.
		return nil, fmt.Errorf("UTMP record with unexpected type %v in bad login file", utmp.UtType)
	}

	return &record, nil
}

// processGoodLoginRecord receives UTMP login records in order and returns
// a corresponding LoginRecord. Some UTMP records do not translate
// into a LoginRecord, in this case the return value is nil.
func (r *UtmpFileReader) processGoodLoginRecord(utmp *Utmp) *LoginRecord {
	record := LoginRecord{
		Utmp:      utmp,
		Timestamp: utmp.UtTv,
		UID:       -1,
		PID:       -1,
	}

	if utmp.UtLine != "~" {
		record.TTY = utmp.UtLine
	}

	switch utmp.UtType {
	// See utmp(5) for C constants.
	case RUN_LVL:
		// The runlevel - though a number - is stored as
		// the ASCII character of that number.
		runlevel := string(rune(utmp.UtPid))

		// 0 - halt; 6 - reboot
		if utmp.UtUser == "shutdown" || runlevel == "0" || runlevel == "6" {
			record.Type = shutdownRecord

			// Clear any old logins
			// TODO: Issue logout events for login events that are still around
			// at this point.
			r.loginSessions = make(map[string]LoginRecord)
		} else {
			// Ignore runlevel changes that are not halt or reboot.
			return nil
		}
	case BOOT_TIME:
		if utmp.UtLine == "~" && utmp.UtUser == "reboot" {
			record.Type = bootRecord

			// Clear any old logins
			// TODO: Issue logout events for login events that are still around
			// at this point.
			r.loginSessions = make(map[string]LoginRecord)
		} else {
			// Ignore unknown record
			return nil
		}
	case USER_PROCESS:
		record.Type = userLoginRecord

		record.Username = utmp.UtUser
		record.UID = lookupUsername(record.Username)
		record.PID = utmp.UtPid
		record.IP = newIP(utmp.UtAddrV6)
		record.Hostname = utmp.UtHost

		// Store TTY from user login record for enrichment when user logout
		// record comes along (which, alas, does not contain the username).
		r.loginSessions[record.TTY] = record
	case DEAD_PROCESS:
		savedRecord, found := r.loginSessions[record.TTY]
		if found {
			record.Type = userLogoutRecord
			record.Username = savedRecord.Username
			record.UID = savedRecord.UID
			record.PID = savedRecord.PID
			record.IP = savedRecord.IP
			record.Hostname = savedRecord.Hostname
		} else {
			// Skip - this is usually the DEAD_PROCESS event for
			// a previous INIT_PROCESS or LOGIN_PROCESS event -
			// those are ignored - (see default case below).
			return nil
		}
	default:
		/*
			Every other record type is ignored:
			- EMPTY - empty record
			- NEW_TIME and OLD_TIME - could be useful, but not written when time changes,
			  at least not using `date`
			- INIT_PROCESS and LOGIN_PROCESS - written on boot but do not contain any
			  interesting information
			- ACCOUNTING - not implemented according to manpage
		*/
		r.log.Debugf("Ignoring UTMP record of type %v.", utmp.UtType)
		return nil
	}

	return &record
}

// lookupUsername looks up a username and returns its UID.
// It does not pass through errors (e.g. when the user is not found)
// but will return -1 instead.
func lookupUsername(username string) int {
	if username != "" {
		user, err := user.Lookup(username)
		if err == nil {
			uid, err := strconv.Atoi(user.Uid)
			if err == nil {
				return uid
			}
		}
	}

	return -1
}

func newIP(utAddrV6 [4]uint32) *net.IP {
	var ip net.IP

	// See utmp(5) for the utmp struct fields.
	if utAddrV6[1] != 0 || utAddrV6[2] != 0 || utAddrV6[3] != 0 {
		// IPv6
		b := make([]byte, 16)
		byteOrder.PutUint32(b[:4], utAddrV6[0])
		byteOrder.PutUint32(b[4:8], utAddrV6[1])
		byteOrder.PutUint32(b[8:12], utAddrV6[2])
		byteOrder.PutUint32(b[12:], utAddrV6[3])
		ip = net.IP(b)
	} else {
		// IPv4
		b := make([]byte, 4)
		byteOrder.PutUint32(b, utAddrV6[0])
		ip = net.IP(b)
	}

	return &ip
}

func (r *UtmpFileReader) saveStateToDisk() error {
	err := r.saveFileRecordsToDisk()
	if err != nil {
		return err
	}

	err = r.saveLoginSessionsToDisk()
	if err != nil {
		return err
	}

	return nil
}

func (r *UtmpFileReader) saveFileRecordsToDisk() error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	for _, utmpFile := range r.savedUtmpFiles {
		err := encoder.Encode(utmpFile)
		if err != nil {
			return fmt.Errorf("error encoding UTMP file record: %w", err)
		}
	}

	err := r.bucket.Store(bucketKeyFileRecords, buf.Bytes())
	if err != nil {
		return fmt.Errorf("error writing UTMP file records to disk: %w", err)
	}

	r.log.Debugf("Wrote %d UTMP file records to disk", len(r.savedUtmpFiles))
	return nil
}

func (r *UtmpFileReader) saveLoginSessionsToDisk() error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	for _, loginRecord := range r.loginSessions {
		err := encoder.Encode(loginRecord)
		if err != nil {
			return fmt.Errorf("error encoding login record: %w", err)
		}
	}

	err := r.bucket.Store(bucketKeyLoginSessions, buf.Bytes())
	if err != nil {
		return fmt.Errorf("error writing login records to disk: %w", err)
	}

	r.log.Debugf("Wrote %d open login sessions to disk", len(r.loginSessions))
	return nil
}

func (r *UtmpFileReader) restoreStateFromDisk() error {
	err := r.restoreFileRecordsFromDisk()
	if err != nil {
		return err
	}

	err = r.restoreLoginSessionsFromDisk()
	if err != nil {
		return err
	}

	return nil
}

func (r *UtmpFileReader) restoreFileRecordsFromDisk() error {
	var decoder *gob.Decoder
	err := r.bucket.Load(bucketKeyFileRecords, func(blob []byte) error {
		if len(blob) > 0 {
			buf := bytes.NewBuffer(blob)
			decoder = gob.NewDecoder(buf)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if decoder != nil {
		for {
			var utmpFile UtmpFile
			err = decoder.Decode(&utmpFile)
			if err == nil {
				r.savedUtmpFiles[utmpFile.Inode] = utmpFile
			} else if err == io.EOF {
				// Read all
				break
			} else {
				return fmt.Errorf("error decoding file record: %w", err)
			}
		}
	}
	r.log.Debugf("Restored %d UTMP file records from disk", len(r.savedUtmpFiles))

	return nil
}

func (r *UtmpFileReader) restoreLoginSessionsFromDisk() error {
	var decoder *gob.Decoder
	err := r.bucket.Load(bucketKeyLoginSessions, func(blob []byte) error {
		if len(blob) > 0 {
			buf := bytes.NewBuffer(blob)
			decoder = gob.NewDecoder(buf)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if decoder != nil {
		for {
			loginRecord := new(LoginRecord)
			err = decoder.Decode(loginRecord)
			if err == nil {
				r.loginSessions[loginRecord.TTY] = *loginRecord
			} else if err == io.EOF {
				// Read all
				break
			} else {
				return fmt.Errorf("error decoding login record: %w", err)
			}
		}
	}
	r.log.Debugf("Restored %d open login sessions from disk", len(r.loginSessions))

	return nil
}
