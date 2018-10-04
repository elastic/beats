// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package memlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type diskStore struct {
	home        string
	logFileName string
	dataFiles   []dataFileInfo // list of all on disk data files

	txid uint64

	logFile *os.File
	logBuf  *bufio.Writer

	fileMode   os.FileMode // Permissions to apply on the Registry File
	bufferSize int

	logEntries       uint
	logInvalid       bool
	logNeedsTruncate bool
}

type diskLoader struct {
	typeConv *typeConv
}

type logAction struct {
	Op string `json:"op"`
}

type storeEntry struct {
	Key    string        `struct:"_key"`
	Fields common.MapStr `struct:",inline"`
}

type dataFileInfo struct {
	path string
	txid uint64
}

type storeMeta struct {
	Version uint `struct:"version"`
}

const (
	logFileName  = "log.json"
	metaFileName = "meta.json"

	storeVersion = 1

	keyField = "_key"
)

func newDiskStore(
	home string,
	dataFiles []dataFileInfo,
	txid uint64,
	mode os.FileMode,
	entries uint,
	logInvalid bool,
	bufferSize uint,
) *diskStore {
	s := &diskStore{
		home:             home,
		logFileName:      filepath.Join(home, logFileName),
		txid:             txid,
		fileMode:         mode,
		logEntries:       entries,
		logInvalid:       logInvalid,
		logNeedsTruncate: false, // only truncate on next checkpoint
		dataFiles:        dataFiles,
		bufferSize:       int(bufferSize),
	}

	s.tryOpenLog()
	return s
}

func (s *diskStore) tryOpenLog() {
	flags := os.O_RDWR | os.O_CREATE
	if s.logNeedsTruncate {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(s.logFileName, flags, s.fileMode)
	if err != nil {
		logp.Err("Failed to open file %v: %v", s.logFileName, err)
		return
	}

	_, err = f.Seek(0, os.SEEK_END)
	if err != nil {
		f.Close()
		return
	}

	if s.logNeedsTruncate {
		s.logEntries = 0 // reset counter if file was truncated on Open
	}

	s.logNeedsTruncate = false
	s.logFile = f
	s.logBuf = bufio.NewWriterSize(&ensureWriter{s.logFile}, s.bufferSize)
}

func (s *diskStore) Close() error {
	if s.logFile != nil {
		err := s.logFile.Close()
		s.logFile = nil
		s.logBuf = nil
		return err
	}

	return nil
}

func (s *diskStore) mustCheckpoint() bool {
	if s.logInvalid {
		// log was not 'complete' on last open or most recent transaction failed when writing the log file
		// -> force checkpoint and clear log file
		return true
	}

	// Ensure log file is open. If log file can not be opened, we fall back to
	// checkpointing.
	if s.logFile == nil {
		s.tryOpenLog()

		if s.logFile == nil {
			return true
		}
	}

	return false
}

func (s *diskStore) numLogs() uint {
	return s.logEntries
}

// commitCheckpoint finishes a transaction by writing a new registry data file.
// The new registry it's name will be `<txid>.json`.
//
// The txid will be incremented upon success.
//
// 1. Create new checkpoint file
//   - create/truncate and open tmp file
//   - Serialize registry table to temporary file
//   - fsync temporary checkpoint file
//   - close temporary checkpoint file
//   - Rename temporary checkpoint file to new `<txid>.json` data file
//   - fsync on store directory
// 2. Truncate transaction log
// 3. Update 'active.json' symlink to most recent data file:
//   - create temporary symlink file pointing to new `<txid>.json>` data file
//   - rename temporary symlink file to `active.json`
//   - fsync on store directory
// 4. Remove old data files
func (s *diskStore) commitCheckpoint(tbl *hashtable) error {
	tmpPath, err := s.checkpointTmpFile(filepath.Join(s.home, "checkpoint"), tbl)
	if err != nil {
		return err
	}

	fileTxID := s.txid + 1
	fileName := fmt.Sprintf("%v.json", fileTxID)
	checkpointPath := filepath.Join(s.home, fileName)

	if err := os.Rename(tmpPath, checkpointPath); err != nil {
		return err
	}
	s.syncHome()

	// clear transaction log once finished
	s.checkpointClearLog()

	// finish current on-disk transaction by increasing the txid
	s.txid++

	s.dataFiles = append(s.dataFiles, dataFileInfo{
		path: checkpointPath,
		txid: fileTxID,
	})

	// delete old transaction files
	if err := s.updateActiveSymLink(); err == nil {
		s.removeOldDataFiles()
	}

	s.syncHome()
	return nil
}

func (s *diskStore) updateActiveSymLink() error {
	activeLink := filepath.Join(s.home, "active.json")

	active, _ := activeDataFile(s.dataFiles)
	if active == "" {
		os.Remove(activeLink) // try, remove active symlink if present.
		return nil
	}

	active = filepath.Base(active)
	tmpLink := filepath.Join(s.home, "active.json.tmp")
	if err := os.Symlink(active, tmpLink); err != nil {
		return err
	}

	err := os.Rename(tmpLink, activeLink)
	if err != nil {
		return err
	}

	s.syncHome()
	return nil
}

func (s *diskStore) syncHome() {
	trySyncPath(s.home)
}

func (s *diskStore) removeOldDataFiles() {
	L := len(s.dataFiles)
	if L <= 1 {
		return
	}

	removable, keep := s.dataFiles[:L-1], s.dataFiles[L-1:]
	for i := range removable {
		path := removable[i].path
		err := os.Remove(path)
		if err == nil || os.IsNotExist(err) {
			continue
		}

		// ohoh... stop removing and construct new array of leftover data files
		s.dataFiles = append(removable[i:], keep...)
		return
	}
	s.dataFiles = keep
}

func (s *diskStore) checkpointTmpFile(baseName string, tbl *hashtable) (string, error) {
	tempfile := baseName + ".new"
	f, err := os.OpenFile(tempfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_SYNC, s.fileMode)
	if err != nil {
		return "", err
	}

	ok := false
	defer func() {
		if !ok {
			f.Close()
		}
	}()

	writer := bufio.NewWriterSize(&ensureWriter{f}, s.bufferSize)
	enc := newJSONEncoder(writer)
	if _, err = writer.Write([]byte{'['}); err != nil {
		return "", err
	}

	first := true
	for _, bin := range tbl.bins {
		for i := range bin {
			entry := &bin[i]

			prefix := []byte(",\n")
			if first {
				prefix = prefix[1:]
				first = false
			}
			if _, err = writer.Write(prefix); err != nil {
				return "", err
			}

			err = enc.Encode(storeEntry{
				Key:    unsafeString(entry.key),
				Fields: entry.value,
			})
			if err != nil {
				return "", err
			}
		}
	}

	if _, err = writer.Write([]byte("\n]")); err != nil {
		return "", err
	}

	if err = writer.Flush(); err != nil {
		return "", err
	}

	if err = syncFile(f); err != nil {
		return "", err
	}

	ok = true
	if err = f.Close(); err != nil {
		return "", err
	}

	return tempfile, nil
}

func (s *diskStore) checkpointClearLog() {
	if s.logFile == nil {
		s.logNeedsTruncate = true
		return
	}

	err := s.logFile.Truncate(0)
	if err == nil {
		_, err = s.logFile.Seek(0, os.SEEK_SET)
	}

	if err != nil {
		s.logFile.Close()
		s.logFile = nil
		s.logBuf = nil
		s.logNeedsTruncate = true
		s.logInvalid = true
	}

	s.logEntries = 0
}

// commitOps finishes a transaction by appending operations to the transaction
// log file.
// A transaction has a `Begin` and `Commit` operation, including a transaction
// IDs. On error a `Rollback` operation will be inserted.
//
// If an error was encountered, that will leave the transaction log in a
// potential broken state, we force a checkpoint based commit within the next
// transaction. The current transaction will be marked as failed.
//
// The txid will be incremented upon success.
func (s *diskStore) commitOps(st *txState) error {
	writer := s.logBuf
	enc := json.NewEncoder(writer)

	ok := false
	defer func() {
		if !ok {
			// writing to log file failed -> fail current transaction and force full
			// checkpoint on next transaction
			s.logInvalid = true
		}
	}()

	err := encOp(enc, &opBegin{ID: s.txid + 1})
	if err != nil {
		return err
	}

	var total uint
	for _, bin := range st.bins {
		for i := range bin {
			entry := &bin[i]
			if !entry.modified {
				continue
			}

			for _, op := range entry.ops {
				if err = encOp(enc, op); err != nil {
					return nil
				}
			}

			total += uint(len(entry.ops))
		}
	}

	err = encOp(enc, &opCommit{ID: s.txid + 1})
	if err != nil {
		return err
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	ok = true
	s.txid++
	s.logEntries += total
	return nil
}

func newDiskLoader() *diskLoader {
	return &diskLoader{
		typeConv: newTypeConv(),
	}
}

func (l *diskLoader) IsReadonly() bool       { return false }
func (l *diskLoader) checkRead() error       { return nil }
func (l *diskLoader) checkWrite() error      { return nil }
func (l *diskLoader) getTypeConv() *typeConv { return l.typeConv }

// loadDataFile create a new hashtable with all key/value pairs found.
func loadDataFile(path string) (hashtable, error) {
	var tbl hashtable
	tbl.init()

	if path == "" {
		return tbl, nil
	}

	hashFn := newHashFn()
	err := readDataFile(path, func(key []byte, state common.MapStr) {
		tbl.set(hashFn(key), key, state)
	})
	return tbl, err
}

// loadLogFile applies all recorded transaction to an already initialized
// memStore.
// The txid is the transaction ID of the last known valid data file.
// Transactions older then txid will be ignored.
// loadLogFile returns the last commited txid in logTxid and the total number
// of operations in logCount.
// An incomplete transaction is recorded at the end of the log file, if
// complete is false.
func loadLogFile(
	store *memStore,
	txid uint64,
	home string,
) (logTxid uint64, logCount uint, complete bool, err error) {
	var tx *memTx
	var entries uint

	err = readLogFile(home, func(rawOp op) error {
		if tx != nil {
			switch op := rawOp.(type) {
			case *opInsertWith:
				entries++
				tx.Set(unsafeKeyRef(op.K), op.V)

			case *opUpdate:
				entries++
				tx.Update(unsafeKeyRef(op.K), op.V)

			case *opRemove:
				entries++
				tx.Remove(unsafeKeyRef(op.K))

			case *opBegin:
				// current transaction is incomplete -> stop processing any ops
				tx.Rollback()
				tx = nil
				return errTxIncomplete

			case *opRollback:
				if op.ID != txid+1 {
					return fmt.Errorf("rollback id %v != transaction id %v", op.ID, txid+1)
				}
				tx.Rollback()
				tx = nil

			case *opCommit:
				if op.ID != txid+1 {
					return fmt.Errorf("commit id %v != transaction id %v", op.ID, txid+1)
				}

				tx.Commit()
				txid++
				tx = nil
			}

			return nil
		}

		op, ok := rawOp.(*opBegin)
		if !ok {
			// ignore all operations if transaction is not active
			return nil
		}

		if op.ID != txid+1 {
			// ignore transaction that don't follow the current transaction
			return nil
		}

		// new transaction
		tx = &memTx{}
		st := &txState{}
		st.init()
		tx.init(newDiskLoader(), store, st)
		return nil
	})
	if err != nil {
		return txid, entries, false, err
	}

	complete = tx == nil
	if !complete {
		// rollback pending changes if current transaction has not been comitted
		tx.Rollback()
		tx = nil
	}

	return txid, entries, complete, err
}

func readDataFile(path string, fn func([]byte, common.MapStr)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var states []map[string]interface{}
	dec := json.NewDecoder(f)
	if err := dec.Decode(&states); err != nil {
		return fmt.Errorf("corrupted data file: %v", err)
	}

	for _, state := range states {
		key := []byte(state[keyField].(string))
		delete(state, keyField)
		fn(key, state)
	}

	return nil
}

// readLogFile iterates all operations found in the transaction log.
func readLogFile(home string, fn func(op) error) error {
	path := filepath.Join(home, logFileName)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for dec.More() {
		var act logAction
		if err := dec.Decode(&act); err != nil {
			return err
		}

		var op op
		switch act.Op {
		case opValInsert:
			op = &opInsertWith{}
		case opValUpdate:
			op = &opUpdate{}
		case opValRemove:
			op = &opRemove{}
		case opValBegin:
			op = &opBegin{}
		case opValCommit:
			op = &opCommit{}
		case opValRollback:
			op = &opRollback{}
		}

		if err := dec.Decode(op); err != nil {
			return err
		}

		if err := fn(op); err != nil {
			return err
		}
	}

	return nil
}

func writeMetaFile(home string, mode os.FileMode) error {
	path := filepath.Join(home, metaFileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	ok := false
	defer func() {
		if !ok {
			f.Close()
		}
	}()

	enc := newJSONEncoder(&ensureWriter{f})
	err = enc.Encode(storeMeta{
		Version: storeVersion,
	})
	if err != nil {
		return err
	}

	if err := syncFile(f); err != nil {
		return err
	}

	ok = true
	if err := f.Close(); err != nil {
		return err
	}

	trySyncPath(home)
	return nil
}

func readMetaFile(home string) (storeMeta, error) {
	var meta storeMeta
	path := filepath.Join(home, metaFileName)

	f, err := os.Open(path)
	if err != nil {
		return meta, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	if err := dec.Decode(&meta); err != nil {
		return meta, fmt.Errorf("can not read store meta file: %v", err)
	}

	return meta, nil
}

func checkMeta(meta storeMeta) error {
	if meta.Version != storeVersion {
		return fmt.Errorf("store version %v not supported", meta.Version)
	}

	return nil
}

// listDataFiles returns a sorted list of data files with txid per file.
// The list is sorted by txid, in ascending order (taking integer overflows
// into account).
func listDataFiles(home string) ([]dataFileInfo, error) {
	files, err := filepath.Glob(filepath.Join(home, "*.json"))
	if err != nil {
		return nil, err
	}

	var infos []dataFileInfo
	for i := range files {

		info, err := os.Lstat(files[i])
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}

		name := filepath.Base(files[i])
		name = name[:len(name)-5] // remove '.json' extension

		id, err := strconv.ParseUint(name, 10, 64)
		if err == nil {
			infos = append(infos, dataFileInfo{
				path: files[i],
				txid: id,
			})
		}
	}

	// empty or most recent snapshot was complete (old data file has been deleted)
	if len(infos) <= 1 {
		return infos, nil
	}

	// sort files by transaction ID
	sort.Slice(infos, func(i, j int) bool {
		idI := infos[i].txid
		idJ := infos[j].txid
		return int64(idI-idJ) < 0 // check idI < idJ (ids can overflow)
	})
	return infos, nil
}

// activeDataFile returns the most recent data file in a list of present (sorted)
// data files.
func activeDataFile(infos []dataFileInfo) (string, uint64) {
	if len(infos) == 0 {
		return "", 0
	}

	active := infos[len(infos)-1]
	return active.path, active.txid
}

func encOp(enc *json.Encoder, op op) error {
	if err := enc.Encode(logAction{Op: op.name()}); err != nil {
		return err
	}
	return enc.Encode(op)
}
