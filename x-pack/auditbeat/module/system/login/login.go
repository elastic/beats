// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package login

import (
	"bytes"
	"encoding/gob"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

const (
	moduleName    = "system"
	metricsetName = "login"

	bucketName           = "auditbeat.login.v1"
	bucketKeyFileRecords = "file_records"
	bucketKeyTTYLookup   = "tty_lookup"

	eventTypeEvent = "event"
)

type FileRecord struct {
	Inode           uint64
	LastCtime       time.Time
	LastLoginRecord LoginRecord
}

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects login records from /var/log/wtmp.
type MetricSet struct {
	mb.BaseMetricSet
	config      Config
	bucket      datastore.Bucket
	log         *logp.Logger
	paths       []string
	fileRecords map[uint64]FileRecord
	ttyLookup   map[string]*LoginRecord
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open persistent datastore")
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(metricsetName),
		bucket:        bucket,
		fileRecords:   make(map[uint64]FileRecord),
		ttyLookup:     make(map[string]*LoginRecord),
	}

	ms.paths, err = filepath.Glob(config.WtmpFilePattern)
	if err != nil {
		return nil, errors.Wrap(err, "failed to expand file pattern")
	}
	// Sort paths in reverse order (oldest/most-rotated file first)
	sort.Sort(sort.Reverse(sort.StringSlice(ms.paths)))
	ms.log.Debugf("Reading files: %v", ms.paths)

	// Load state (file records, tty mapping) from disk
	err = ms.restoreStateFromDisk()
	if err != nil {
		return nil, errors.Wrap(err, "failed to restore state from disk")
	}

	return ms, nil
}

// Close cleans up the MetricSet when it finishes.
func (ms *MetricSet) Close() error {
	if ms.bucket != nil {
		return ms.bucket.Close()
	}
	return nil
}

// Fetch collects any new login records from /var/log/wtmp. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	// Store inodes still in use for clean up at the end
	currentInodes := make(map[uint64]struct{}, len(ms.fileRecords))

	var newRecords []LoginRecord

	for _, path := range ms.paths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				// Skip - file might have been rotated out
				ms.log.Debugf("File %v does not exist anymore.", path)
			} else {
				ms.log.Error(err)
				report.Error(err)
			}
			continue // Don't stop because of just one file
		}

		statsI := fileInfo.Sys()
		if statsI == nil {
			ms.log.Error(err)
			report.Error(err)
			return
		}
		stats := statsI.(*syscall.Stat_t)
		currentInodes[stats.Ino] = struct{}{}
		ctime := time.Unix(stats.Ctim.Sec, stats.Ctim.Nsec)

		fileRecord, isKnownFile := ms.fileRecords[stats.Ino]
		if !isKnownFile || fileRecord.LastCtime.Before(ctime) {
			loginRecords, err := ReadUtmpFile(ms.log, path)
			if err != nil {
				ms.log.Error(err)
				report.Error(err)
				return
			}

			// When we've read the file before, we want to read through its
			// entries until we reach the last known record, then start
			// emitting everything after that.
			reachedNewRecords := !isKnownFile
			for _, loginRecord := range loginRecords {
				if reachedNewRecords {
					ms.processLoginRecord(&loginRecord)
					newRecords = append(newRecords, loginRecord)
				}

				if isKnownFile && loginRecord.Hash() == fileRecord.LastLoginRecord.Hash() {
					reachedNewRecords = true
				}
			}

			ms.fileRecords[stats.Ino] = FileRecord{
				Inode:           stats.Ino,
				LastCtime:       ctime,
				LastLoginRecord: loginRecords[len(loginRecords)-1],
			}
		}
	}

	// Clean up old file records (where the inode no longer exists)
	for inode, _ := range ms.fileRecords {
		if _, found := currentInodes[inode]; !found {
			ms.log.Debugf("Deleting old inode %d", inode)
			delete(ms.fileRecords, inode)
		}
	}

	// Emit events
	for _, loginRecord := range newRecords {
		report.Event(ms.loginEvent(loginRecord))
	}

	// Save latest read records to disk
	if len(newRecords) > 0 {
		err := ms.saveStateToDisk()
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
			return
		}
	}
}

func (ms *MetricSet) processLoginRecord(record *LoginRecord) {
	switch record.Type {
	case UserLogin:
		// Store TTY from user login record for enrichment when user logout
		// record comes along (which, alas, does not contain the username).
		ms.ttyLookup[record.TTY] = record
	case UserLogout:
		savedRecord, found := ms.ttyLookup[record.TTY]
		if found {
			record.Username = savedRecord.Username
		} else {
			ms.log.Debugf("No matching login record found for logout on %v", record.TTY)
		}
	}
}

func (ms *MetricSet) saveStateToDisk() error {
	err := ms.saveFileRecordsToDisk()
	if err != nil {
		return err
	}

	err = ms.saveTTYLookupToDisk()
	if err != nil {
		return err
	}

	return nil
}

func (ms *MetricSet) saveFileRecordsToDisk() error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	for _, fileRecord := range ms.fileRecords {
		err := encoder.Encode(fileRecord)
		if err != nil {
			return errors.Wrap(err, "error encoding file record")
		}
	}

	err := ms.bucket.Store(bucketKeyFileRecords, buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "error writing file records to disk")
	}

	ms.log.Debugf("Wrote %d file records to disk", len(ms.fileRecords))
	return nil
}

func (ms *MetricSet) saveTTYLookupToDisk() error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	for _, loginRecord := range ms.ttyLookup {
		err := encoder.Encode(*loginRecord)
		if err != nil {
			return errors.Wrap(err, "error encoding login record")
		}
	}

	err := ms.bucket.Store(bucketKeyTTYLookup, buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "error writing login records to disk")
	}

	ms.log.Debugf("Wrote %d open login records to disk", len(ms.ttyLookup))
	return nil
}

func (ms *MetricSet) restoreStateFromDisk() error {
	err := ms.restoreFileRecordsFromDisk()
	if err != nil {
		return err
	}

	err = ms.restoreTTYLookupFromDisk()
	if err != nil {
		return err
	}

	return nil
}

func (ms *MetricSet) restoreFileRecordsFromDisk() error {
	var decoder *gob.Decoder
	err := ms.bucket.Load(bucketKeyFileRecords, func(blob []byte) error {
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
			fileRecord := new(FileRecord)
			err = decoder.Decode(fileRecord)
			if err == nil {
				ms.fileRecords[fileRecord.Inode] = *fileRecord
			} else if err == io.EOF {
				// Read all
				break
			} else {
				return errors.Wrap(err, "error decoding file record")
			}
		}
	}
	ms.log.Debugf("Restored %d file records from disk", len(ms.fileRecords))

	return nil
}

func (ms *MetricSet) restoreTTYLookupFromDisk() error {
	var decoder *gob.Decoder
	err := ms.bucket.Load(bucketKeyTTYLookup, func(blob []byte) error {
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
				ms.ttyLookup[loginRecord.TTY] = loginRecord
			} else if err == io.EOF {
				// Read all
				break
			} else {
				return errors.Wrap(err, "error decoding login record")
			}
		}
	}
	ms.log.Debugf("Restored %d open login records from disk", len(ms.ttyLookup))

	return nil
}

func (ms *MetricSet) loginEvent(loginRecord LoginRecord) mb.Event {
	event := mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"type":   eventTypeEvent,
				"action": loginRecord.Type.String(),
			},
		},
		MetricSetFields: loginRecord.toMapStr(),
	}

	if loginRecord.Username != "" {
		event.RootFields.Put("user", common.MapStr{
			"name": loginRecord.Username,
		})

		user, err := user.Lookup(loginRecord.Username)
		if err == nil {
			uid, err := strconv.ParseUint(user.Uid, 10, 32)
			if err == nil {
				event.MetricSetFields.Put("user.uid", uid)
				event.RootFields.Put("user.id", uid)
			}
		}
	}

	return event
}
