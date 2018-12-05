// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,cgo

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

	bucketName = "auditbeat.login.v1"

	eventTypeEvent = "event"

	// LoginRecordTypeBoot represents a system boot.
	LoginRecordTypeBoot = "boot"
	// LoginRecordTypeShutdown represents a system shutdown (halt or reboot).
	LoginRecordTypeShutdown = "shutdown"
	// LoginRecordTypeUserLogin represents a user login.
	LoginRecordTypeUserLogin = "user_login"
	// LoginRecordTypeUserLogout represents a user logout.
	LoginRecordTypeUserLogout = "user_logout"
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
	mapstr := common.MapStr{}

	// Very useful for development
	//mapstr.Put("utmp", fmt.Sprintf("%v", login.Utmp))

	if login.TTY != "" {
		mapstr.Put("tty", login.TTY)
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

	ms.utmpReader, err = NewUtmpFileReader(ms.log, bucket, config.UtmpFilePattern)
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

	// Save new state to disk
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
		event.RootFields.Put("user.name", loginRecord.Username)

		if loginRecord.UID != -1 {
			event.RootFields.Put("user.id", loginRecord.UID)
		}
	}

	if loginRecord.PID != -1 {
		event.RootFields.Put("process.pid", loginRecord.PID)
	}

	if loginRecord.IP != nil {
		event.RootFields.Put("source.ip", loginRecord.IP)
	}

	if loginRecord.Hostname != "" && loginRecord.Hostname != loginRecord.IP.String() {
		event.RootFields.Put("source.domain", loginRecord.Hostname)
	}

	var message string

	switch loginRecord.Type {
	case LoginRecordTypeBoot:
		message = "System boot"
	case LoginRecordTypeShutdown:
		message = "System shutdown"
	case LoginRecordTypeUserLogin:
		message = fmt.Sprintf("Login by user %v (UID: %d) on %v (PID: %d) from %v (IP: %v).",
			loginRecord.Username, loginRecord.UID, loginRecord.TTY, loginRecord.PID,
			loginRecord.Hostname, loginRecord.IP)
	case LoginRecordTypeUserLogout:
		message = fmt.Sprintf("Logout by user %v (UID: %d) on %v (PID: %d) from %v (IP: %v).",
			loginRecord.Username, loginRecord.UID, loginRecord.TTY, loginRecord.PID,
			loginRecord.Hostname, loginRecord.IP)
	}

	event.RootFields.Put("message", message)

	return event
}
