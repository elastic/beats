// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// MaxSplay is the maximum allowed splay duration (12 hours)
	MaxSplay = 12 * time.Hour
	// DefaultSplay is the default splay duration (disabled)
	DefaultSplay = 0
)

// RRuleScheduleConfig represents an RRULE-based schedule configuration
// This provides an alternative to osquery's native interval-based scheduling
type RRuleScheduleConfig struct {
	// RRule is the RFC 5545 recurrence rule string
	// Examples: "FREQ=DAILY", "FREQ=WEEKLY;BYDAY=MO,WE"
	RRule string `config:"rrule" json:"rrule,omitempty"`

	// StartDate is the required start date for the schedule (RFC3339 format)
	// Queries will not run before this date
	StartDate string `config:"start_date,omitempty" json:"start_date,omitempty"`

	// EndDate is the optional end date for the schedule (RFC3339 format)
	// Queries will not run after this date
	EndDate string `config:"end_date,omitempty" json:"end_date,omitempty"`

	// Splay is the maximum random delay before query execution.
	// This helps spread out query execution times to avoid thundering herd effects.
	// Accepts duration strings: "30s", "5m", "2h", etc.
	// Range: 0s to 12h. Default: 0s (disabled).
	Splay string `config:"splay,omitempty" json:"splay,omitempty"`

	// Timeout is the query execution timeout in seconds
	// Default is 60 seconds if not specified
	Timeout int `config:"timeout,omitempty" json:"timeout,omitempty"`
}

// GetSplay parses and returns the splay duration, defaulting to 0s if not set
func (c *RRuleScheduleConfig) GetSplay() (time.Duration, error) {
	if c.Splay == "" {
		return DefaultSplay, nil
	}

	d, err := time.ParseDuration(c.Splay)
	if err != nil {
		return 0, fmt.Errorf("invalid splay duration '%s': %w", c.Splay, err)
	}

	if d < 0 {
		return 0, fmt.Errorf("splay cannot be negative: %s", c.Splay)
	}

	if d > MaxSplay {
		return 0, fmt.Errorf("splay cannot exceed %v, got: %s", MaxSplay, c.Splay)
	}

	return d, nil
}

// ParseStartDate parses the start date string into a time.Time pointer
func (c *RRuleScheduleConfig) ParseStartDate() (*time.Time, error) {
	if c.StartDate == "" {
		return nil, fmt.Errorf("start_date is required for rrule schedules")
	}
	t, err := time.Parse(time.RFC3339, c.StartDate)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ParseEndDate parses the end date string into a time.Time pointer
func (c *RRuleScheduleConfig) ParseEndDate() (*time.Time, error) {
	if c.EndDate == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, c.EndDate)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// IsEnabled returns true if an RRULE is configured
func (c *RRuleScheduleConfig) IsEnabled() bool {
	return c != nil && c.RRule != ""
}

// NativeSchedule holds interval and policy fields for native (interval-based) schedules.
// Embedded in Query so these appear at the same level in config (query, interval, schedule_id, start_date).
type NativeSchedule struct {
	Interval   int    `config:"interval" json:"interval"`
	ScheduleID string `config:"schedule_id,omitempty" json:"schedule_id,omitempty"` // from Kibana; used in scheduled result/response docs
	StartDate  string `config:"start_date,omitempty" json:"start_date,omitempty"`  // RFC3339; for schedule_execution_count
}

type Query struct {
	Query          string `config:"query" json:"query"`
	NativeSchedule `config:",inline"` // interval, schedule_id, start_date (flat in config)
	Platform       string `config:"platform" json:"platform,omitempty"`
	Version     string `config:"version" json:"version,omitempty"`
	Shard       int    `config:"shard" json:"shard,omitempty"`
	Description int    `config:"description" json:"description,omitempty"`

	// Optional ECS mapping for the query, not rendered into osqueryd configuration
	ECSMapping map[string]interface{} `config:"ecs_mapping" json:"-"`

	// A boolean to set 'snapshot' mode, default true
	// This is different from the default osquery behavior where the missing value defaults to false
	Snapshot *bool `config:"snapshot,omitempty" json:"snapshot,omitempty"`

	// A boolean to determine if "removed" actions should be logged, default true
	// This is the same as osquery behavior
	Removed *bool `config:"removed,omitempty" json:"removed,omitempty"`

	// RRuleSchedule provides RRULE-based scheduling as an alternative to interval
	// When set, queries are scheduled by osquerybeat instead of osqueryd's native scheduler
	// If both interval and rrule_schedule are set, rrule_schedule takes precedence
	RRuleSchedule *RRuleScheduleConfig `config:"rrule_schedule,omitempty" json:"-"`
}

type Pack struct {
	Discovery []string         `config:"discovery" json:"discovery,omitempty"`
	Platform  string           `config:"platform" json:"platform,omitempty"`
	Version   string           `config:"version" json:"version,omitempty"`
	Shard     int              `config:"shard" json:"shard,omitempty"`
	Queries   map[string]Query `config:"queries" json:"queries,omitempty"`
}

// > SELECT * FROM osquery_events where type = 'subscriber';
// +---------------------+---------------------+------------+---------------+--------+-----------+--------+
// | name                | publisher           | type       | subscriptions | events | refreshes | active |
// +---------------------+---------------------+------------+---------------+--------+-----------+--------+
// | apparmor_events     | auditeventpublisher | subscriber | 0             | 0      | 0         | 0      |
// | bpf_process_events  | BPFEventPublisher   | subscriber | 0             | 0      | 0         | 0      |
// | bpf_socket_events   | BPFEventPublisher   | subscriber | 0             | 0      | 0         | 0      |
// | file_events         | inotify             | subscriber | 0             | 0      | 0         | 0      |
// | hardware_events     | udev                | subscriber | 0             | 0      | 0         | 0      |
// | process_events      | auditeventpublisher | subscriber | 0             | 0      | 0         | 0      |
// | process_file_events | auditeventpublisher | subscriber | 0             | 0      | 0         | 0      |
// | seccomp_events      | auditeventpublisher | subscriber | 0             | 0      | 0         | 0      |
// | selinux_events      | auditeventpublisher | subscriber | 0             | 0      | 0         | 0      |
// | socket_events       | auditeventpublisher | subscriber | 0             | 0      | 0         | 0      |
// | syslog_events       | syslog              | subscriber | 0             | 0      | 0         | 0      |
// | user_events         | auditeventpublisher | subscriber | 0             | 0      | 0         | 0      |
// | yara_events         | inotify             | subscriber | 0             | 0      | 0         | 0      |
// +---------------------+---------------------+------------+---------------+--------+-----------+--------+

// The configuration supports a method to explicitly allow and deny events subscribers.
// If you choose to explicitly allow subscribers, then all will be disabled except for those specificied in the allow list.
// If you choose to explicitly deny subscribers, then all will be enabled except for those specificied in the deny list.
type Events struct {
	EnableSubscribers  []string `config:"enable_subscribers" json:"enable_subscribers,omitempty"`
	DisableSubscribers []string `config:"disable_subscribers" json:"disable_subscribers,omitempty"`
}

type OsqueryConfig struct {
	Options               map[string]interface{} `config:"options" json:"options,omitempty"`
	Schedule              map[string]Query       `config:"schedule" json:"schedule,omitempty"`
	Packs                 map[string]Pack        `config:"packs" json:"packs,omitempty"`
	Filepaths             map[string][]string    `config:"file_paths" json:"file_paths,omitempty"`
	Views                 map[string]string      `config:"views" json:"views,omitempty"`
	Events                *Events                `config:"events" json:"events,omitempty"`
	Yara                  map[string]interface{} `config:"yara" json:"yara,omitempty"`
	PrometheusTargets     map[string]interface{} `config:"prometheus_targets" json:"prometheus_targets,omitempty"`
	AutoTableConstruction map[string]interface{} `config:"auto_table_construction" json:"auto_table_construction,omitempty"`
}

// Render returns the full config as JSON (includes policy-only fields for osquerybeat).
func (c OsqueryConfig) Render() ([]byte, error) {
	return json.MarshalIndent(c, "", "    ")
}
