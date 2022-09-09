// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package login

import (
	"fmt"
	"net"
	"time"

	"github.com/elastic/beats/v7/auditbeat/datastore"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	moduleName    = "system"
	metricsetName = "login"
	namespace     = "system.audit.login"

	bucketName = "login.v1"

	eventTypeEvent = "event"
)

// loginRecordType represents the type of a login record.
type loginRecordType uint8

const (
	bootRecord loginRecordType = iota + 1
	shutdownRecord
	userLoginRecord
	userLogoutRecord
	userLoginFailedRecord
)

// String returns the string representation of a LoginRecordType.
func (t loginRecordType) string() string {
	switch t {
	case bootRecord:
		return "boot"
	case shutdownRecord:
		return "shutdown"

	case userLoginFailedRecord:
		fallthrough
	case userLoginRecord:
		return "user_login"

	case userLogoutRecord:
		return "user_logout"
	default:
		return ""
	}
}

// LoginRecord represents a login record.
type LoginRecord struct {
	Utmp      *Utmp
	Type      loginRecordType
	PID       int
	TTY       string
	UID       int
	Username  string
	Hostname  string
	IP        *net.IP
	Timestamp time.Time
	Origin    string
}

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithNamespace(namespace),
	)
}

// MetricSet collects login records from /var/log/wtmp.
type MetricSet struct {
	mb.BaseMetricSet
	config     config
	log        *logp.Logger
	utmpReader *UtmpFileReader
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The %v/%v dataset is beta", moduleName, metricsetName)

	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the %v/%v config: %w", moduleName, metricsetName, err)
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to open persistent datastore: %w", err)
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(metricsetName),
	}

	ms.utmpReader, err = NewUtmpFileReader(ms.log, bucket, config)
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
	count := ms.readAndEmit(report)

	ms.log.Debugf("%d new login records.", count)

	// Save new state to disk
	if count > 0 {
		err := ms.utmpReader.saveStateToDisk()
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
		}
	}
}

// readAndEmit reads and emits login events and returns the number of events.
func (ms *MetricSet) readAndEmit(report mb.ReporterV2) int {
	loginRecordC, errorC := ms.utmpReader.ReadNew()

	var count int
	for {
		select {
		case loginRecord, ok := <-loginRecordC:
			if !ok {
				return count
			}
			report.Event(ms.loginEvent(&loginRecord))
			count++
		case err, ok := <-errorC:
			if !ok {
				return count
			}
			ms.log.Error(err)
		}
	}
}

func (ms *MetricSet) loginEvent(loginRecord *LoginRecord) mb.Event {
	event := mb.Event{
		Timestamp: loginRecord.Timestamp,
		RootFields: mapstr.M{
			"event": mapstr.M{
				"kind":   eventTypeEvent,
				"action": loginRecord.Type.string(),
				"origin": loginRecord.Origin,
			},
			"message": loginMessage(loginRecord),
			// Very useful for development
			// "debug": fmt.Sprintf("%v", login.Utmp),
		},
	}

	if loginRecord.Username != "" {
		event.RootFields.Put("user.name", loginRecord.Username)
		event.RootFields.Put("related.user", []string{loginRecord.Username})
		if loginRecord.UID != -1 {
			event.RootFields.Put("user.id", loginRecord.UID)
		}
	}

	if loginRecord.TTY != "" {
		event.RootFields.Put("user.terminal", loginRecord.TTY)
	}

	if loginRecord.PID != -1 {
		event.RootFields.Put("process.pid", loginRecord.PID)
	}

	if loginRecord.IP != nil {
		event.RootFields.Put("source.ip", loginRecord.IP)
		event.RootFields.Put("related.ip", []string{loginRecord.IP.String()})
	}

	if loginRecord.Hostname != "" && loginRecord.Hostname != loginRecord.IP.String() {
		event.RootFields.Put("source.domain", loginRecord.Hostname)
	}

	switch loginRecord.Type {
	case userLoginRecord:
		event.RootFields.Put("event.category", []string{"authentication"})
		event.RootFields.Put("event.outcome", "success")
		event.RootFields.Put("event.type", []string{"start", "authentication_success"})
	case userLoginFailedRecord:
		event.RootFields.Put("event.category", []string{"authentication"})
		event.RootFields.Put("event.outcome", "failure")
		event.RootFields.Put("event.type", []string{"start", "authentication_failure"})
	case userLogoutRecord:
		event.RootFields.Put("event.category", []string{"authentication"})
		event.RootFields.Put("event.type", []string{"end"})
	case bootRecord:
		event.RootFields.Put("event.category", []string{"host"})
		event.RootFields.Put("event.type", []string{"start"})
	case shutdownRecord:
		event.RootFields.Put("event.category", []string{"host"})
		event.RootFields.Put("event.type", []string{"end"})
	}

	return event
}

func loginMessage(loginRecord *LoginRecord) string {
	var actionString string

	switch loginRecord.Type {
	case bootRecord:
		return "System boot"
	case shutdownRecord:
		return "System shutdown"
	case userLoginRecord:
		actionString = "Login"
	case userLoginFailedRecord:
		actionString = "Failed login"
	case userLogoutRecord:
		actionString = "Logout"
	}

	return fmt.Sprintf("%v by user %v (UID: %d) on %v (PID: %d) from %v (IP: %v)",
		actionString, loginRecord.Username, loginRecord.UID, loginRecord.TTY, loginRecord.PID,
		loginRecord.Hostname, loginRecord.IP)
}
