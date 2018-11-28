// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package login

import (
	"bytes"
	"encoding/gob"
	"io"
	"net"
	"os/user"
	"strconv"
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
	Utmp      Utmp
	Type      RecordType
	PID       int
	TTY       string
	UID       int
	Username  string
	Hostname  string
	IP        net.IP
	Timestamp time.Time
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

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects login records from /var/log/wtmp.
type MetricSet struct {
	mb.BaseMetricSet
	config     Config
	bucket     datastore.Bucket
	log        *logp.Logger
	ttyLookup  map[string]*LoginRecord
	utmpReader *UtmpFileReader
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
		ttyLookup:     make(map[string]*LoginRecord),
	}

	ms.utmpReader = &UtmpFileReader{
		log:         ms.log,
		filePattern: config.WtmpFilePattern,
		fileRecords: make(map[uint64]FileRecord),
	}

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
	loginRecords, err := ms.utmpReader.ReadNew()
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
		return
	}
	ms.log.Debugf("Read %d new records.", len(loginRecords))

	for _, loginRecord := range loginRecords {
		ms.processLoginRecord(&loginRecord)
		report.Event(ms.loginEvent(loginRecord))
	}

	// Save latest read records to disk
	if len(loginRecords) > 0 {
		err := ms.saveStateToDisk()
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
			return
		}
	}
}

func (ms *MetricSet) processLoginRecord(record *LoginRecord) {
	user, err := user.Lookup(record.Username)
	if err == nil {
		uid, err := strconv.Atoi(user.Uid)
		if err == nil {
			record.UID = uid
		}
	}

	switch record.Type {
	case UserLogin:
		// Store TTY from user login record for enrichment when user logout
		// record comes along (which, alas, does not contain the username).
		ms.ttyLookup[record.TTY] = record
	case UserLogout:
		savedRecord, found := ms.ttyLookup[record.TTY]
		if found {
			record.Username = savedRecord.Username
			record.UID = savedRecord.UID
			record.IP = savedRecord.IP
			record.Hostname = savedRecord.Hostname
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

	for _, fileRecord := range ms.utmpReader.fileRecords {
		err := encoder.Encode(fileRecord)
		if err != nil {
			return errors.Wrap(err, "error encoding file record")
		}
	}

	err := ms.bucket.Store(bucketKeyFileRecords, buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "error writing file records to disk")
	}

	ms.log.Debugf("Wrote %d file records to disk", len(ms.utmpReader.fileRecords))
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
				ms.utmpReader.fileRecords[fileRecord.Inode] = *fileRecord
			} else if err == io.EOF {
				// Read all
				break
			} else {
				return errors.Wrap(err, "error decoding file record")
			}
		}
	}
	ms.log.Debugf("Restored %d file records from disk", len(ms.utmpReader.fileRecords))

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

		if loginRecord.UID != -1 {
			event.MetricSetFields.Put("user.uid", loginRecord.UID)
			event.RootFields.Put("user.id", loginRecord.UID)
		}
	}

	return event
}
