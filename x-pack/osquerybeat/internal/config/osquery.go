// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"encoding/json"
)

type Query struct {
	Query       string `config:"query" json:"query"`
	Interval    int    `config:"interval" json:"interval"`
	Platform    string `config:"platform" json:"platform,omitempty"`
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

	// ActionID is the policy-defined action identifier for this scheduled query.
	// Stored in the policy and used in result/response documents for correlation.
	// If empty, the query name is used as the action_id when publishing.
	ActionID string `config:"action_id,omitempty" json:"action_id,omitempty"`

	// StartDate is the optional start date for native (interval-based) schedules (RFC3339).
	// Used as the reference for schedule_execution_count. For RRULE schedules, start_date
	// is defined in rrule_schedule instead.
	StartDate string `config:"start_date,omitempty" json:"start_date,omitempty"`
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

	// ScheduleSplayPercent controls the spread of native interval-based scheduled queries
	// This is a percentage (0-100) of the query interval to randomize start times
	// Default is 10%. Set to 0 to disable splay for native queries.
	// Note: This only affects queries using 'interval' (native osquery scheduling).
	// For cron-scheduled queries, use the splay field in cron_schedule instead.
	ScheduleSplayPercent *int `config:"schedule_splay_percent,omitempty" json:"-"`

	// ScheduleMaxDrift is the max time drift in seconds for splay compensation
	// The scheduler tries to compensate for splay drift until the delta exceeds this value.
	// If exceeded, the splay resets to zero and compensation restarts.
	// This prevents endless CPU-intensive compensation after long pauses (SIGSTOP/SIGCONT).
	// Default is 60 seconds. Set to 0 to disable drift compensation.
	// Note: This only affects native osquery scheduling, not cron-scheduled queries.
	ScheduleMaxDrift *int `config:"schedule_max_drift,omitempty" json:"-"`
}

// Render serializes the OsqueryConfig to JSON for osqueryd configuration.
// It applies any first-class config fields (like schedule_splay_percent) to the options map.
func (c OsqueryConfig) Render() ([]byte, error) {
	// Create a copy for rendering to avoid modifying the original
	renderConfig := c

	// Apply schedule_splay_percent to options if set
	if c.ScheduleSplayPercent != nil {
		if renderConfig.Options == nil {
			renderConfig.Options = make(map[string]interface{})
		}
		// Only set if not already explicitly set in options
		if _, exists := renderConfig.Options["schedule_splay_percent"]; !exists {
			renderConfig.Options["schedule_splay_percent"] = *c.ScheduleSplayPercent
		}
	}

	// Apply schedule_max_drift to options if set
	if c.ScheduleMaxDrift != nil {
		if renderConfig.Options == nil {
			renderConfig.Options = make(map[string]interface{})
		}
		// Only set if not already explicitly set in options
		if _, exists := renderConfig.Options["schedule_max_drift"]; !exists {
			renderConfig.Options["schedule_max_drift"] = *c.ScheduleMaxDrift
		}
	}

	return json.MarshalIndent(renderConfig, "", "    ")
}
