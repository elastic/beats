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
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
)

const (
	moduleName    = "system"
	metricsetName = "login"

	bucketName           = "auditbeat.login.v1"
	bucketKeyFileRecords = "login_records"

	eventTypeEvent = "event"

	eventActionLogin = "login"
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
	osFamily    string
	cache       *cache.Cache
	bucket      datastore.Bucket
	log         *logp.Logger
	paths       []string
	fileRecords map[uint64]FileRecord
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	paths, err := filepath.Glob(config.WtmpFilePattern)
	if err != nil {
		return nil, errors.Wrap(err, "failed to expand file pattern")
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open persistent datastore")
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(metricsetName),
		paths:         paths,
		bucket:        bucket,
		fileRecords:   make(map[uint64]FileRecord),
	}

	// Load file records from disk
	err = ms.restoreFileRecordsFromDisk()
	if err != nil {
		return nil, errors.Wrap(err, "failed to restore file records from disk")
	}
	ms.log.Debugf("Restored %d file records from disk", len(ms.fileRecords))

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
			loginRecords, err := ReadUtmpFile(path)
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
		err := ms.saveFileRecordsToDisk()
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
			return
		}
	}
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

	return nil
}

func (ms *MetricSet) loginEvent(loginRecord LoginRecord) mb.Event {
	event := mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"type":   eventTypeEvent,
				"action": eventActionLogin,
			},
			"user": common.MapStr{
				"name": loginRecord.Username,
			},
		},
		MetricSetFields: loginRecord.toMapStr(),
	}

	user, err := user.Lookup(loginRecord.Username)
	if err == nil {
		uid := strconv.ParseUint(user.Uid, 10, 32)
		event.MetricSetFields.Put("uid", uid)
		event.RootFields.Put("user.id", uid)
	}

	return event
}
