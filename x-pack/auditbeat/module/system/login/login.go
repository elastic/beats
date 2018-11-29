// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package login

import (
	"fmt"
	"net"
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

// LoginRecord represents a login record.
type LoginRecord struct {
	Utmp      Utmp
	Type      string
	PID       int
	TTY       string
	UID       int
	Username  string
	Hostname  string
	IP        *net.IP
	Timestamp time.Time
	Origin    string
}

func (login LoginRecord) toMapStr() common.MapStr {
	mapstr := common.MapStr{
		"type": login.Type,
		"utmp": fmt.Sprintf("%v", login.Utmp),
	}

	if login.TTY != "" {
		mapstr.Put("tty", login.TTY)
	}

	if login.PID != -1 {
		mapstr.Put("pid", login.PID)
	}

	if login.Username != "" {
		mapstr.Put("user", common.MapStr{
			"name": login.Username,
		})
	}

	if login.Hostname != "" {
		mapstr.Put("hostname", login.Hostname)
	}

	if login.IP != nil {
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
	log        *logp.Logger
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
	}

	ms.utmpReader, err = NewUtmpFileReader(ms.log, bucket, config.WtmpFilePattern)
	if err != nil {
		return nil, err
	}

	return ms, nil
}

// Close cleans up the MetricSet when it finishes.
func (ms *MetricSet) Close() error {
	return ms.utmpReader.Close()
}

// Fetch collects any new login records from /var/log/wtmp. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	loginRecords, err := ms.utmpReader.ReadNew()
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
		return
	}
	ms.log.Debugf("%d new login records.", len(loginRecords))

	for _, loginRecord := range loginRecords {
		report.Event(ms.loginEvent(loginRecord))
	}

	// Save latest read records to disk
	if len(loginRecords) > 0 {
		err := ms.utmpReader.saveStateToDisk()
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
			return
		}
	}
}

func (ms *MetricSet) loginEvent(loginRecord LoginRecord) mb.Event {
	event := mb.Event{
		Timestamp: loginRecord.Timestamp,
		RootFields: common.MapStr{
			"event": common.MapStr{
				"type":   eventTypeEvent,
				"action": loginRecord.Type,
				"origin": loginRecord.Origin,
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

	var eventSummary string

	switch loginRecord.Type {
	case Boot:
		eventSummary = "System boot"
	case Shutdown:
		eventSummary = "Shutdown"
	case UserLogin:
		eventSummary = fmt.Sprintf("Login by user %v (UID: %d) on %v (PID: %d) from %v (IP: %v).",
			loginRecord.Username, loginRecord.UID, loginRecord.TTY, loginRecord.PID,
			loginRecord.Hostname, loginRecord.IP)
	case UserLogout:
		eventSummary = fmt.Sprintf("Logout by user %v (UID: %d) on %v (PID: %d) from %v (IP: %v).",
			loginRecord.Username, loginRecord.UID, loginRecord.TTY, loginRecord.PID,
			loginRecord.Hostname, loginRecord.IP)
	}

	event.RootFields.Put("event.summary", eventSummary)

	return event
}
