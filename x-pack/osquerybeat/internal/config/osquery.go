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
	// MaxSplay is the maximum allowed splay duration (aligns with daily-or-longer RRULE minimum).
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
	// Range: 0s to 12h (see MaxSplay). Default: 0s (disabled).
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

// ElasticOptions contains Beat-specific options that are not part of
// osquery's native config schema.
type ElasticOptions struct {
	Install             *InstallConfig             `config:"install" json:"-"`
	QueryProfileStorage *QueryProfileStorageConfig `config:"query_profile_storage" json:"-"`
}

// QueryProfileStorageConfig controls local storage of live query profiles.
type QueryProfileStorageConfig struct {
	Enabled     *bool `config:"enabled" json:"-"`
	MaxProfiles int   `config:"max_profiles" json:"-"`
}

func (c QueryProfileStorageConfig) EnabledOrDefault() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

func (c QueryProfileStorageConfig) MaxProfilesOrDefault() int {
	if c.MaxProfiles <= 0 {
		return DefaultQueryProfileMaxProfiles
	}
	return c.MaxProfiles
}

type CommonScheduleConfig struct {
	// SpaceID can match across queries in a pack; Fleet may set a pack-level default_space_id.
	SpaceID string `config:"space_id,omitempty" json:"space_id,omitempty"`
	// ScheduleID is always per-query (policy schedule identity); it is not inherited from the pack.
	ScheduleID string `config:"schedule_id,omitempty" json:"schedule_id,omitempty"`
}

// NativeSchedule holds interval and start_date for native (interval-based) schedules.
// Used for Query (embedded with CommonScheduleConfig) and for Pack.DefaultNativeSchedule.
type NativeSchedule struct {
	Interval  int    `config:"interval" json:"interval,omitempty"`
	StartDate string `config:"start_date,omitempty" json:"start_date,omitempty"` // RFC3339; for schedule_execution_count
}

type Query struct {
	Query string `config:"query" json:"query"`

	CommonScheduleConfig `config:",inline"`
	NativeSchedule       `config:",inline"`
	// RRuleSchedule provides RRULE-based scheduling as an alternative to interval.
	// When set, queries are scheduled by osquerybeat instead of osqueryd's native scheduler.
	// A query must not set both interval (native) and rrule_schedule; see ValidateQueryScheduleMode.
	RRuleSchedule *RRuleScheduleConfig `config:"rrule_schedule,omitempty" json:"-"`

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

	// Optional internal flag to emit per-query profiling for this scheduled query
	// (native: osquery_schedule metrics; RRULE: process-level deltas like live queries).
	// RRULE/live profiling uses the same serialized osquery client; native osqueryd schedules
	// can still add process load outside that bracket. Not rendered into osqueryd configuration.
	Profile bool `config:"profile" json:"-"`
}

type Pack struct {
	// PackID is the policy-defined pack identifier; used in result/response documents for correlation.
	// If empty, the pack map key (pack name) is used when publishing.
	PackID    string   `config:"pack_id,omitempty" json:"pack_id,omitempty"`
	Discovery []string `config:"discovery" json:"discovery,omitempty"`
	Platform  string   `config:"platform" json:"platform,omitempty"`
	Version   string   `config:"version" json:"version,omitempty"`
	Shard     int      `config:"shard" json:"shard,omitempty"`

	// DefaultNativeSchedule provides interval and start_date defaults for queries in this pack
	// that omit them. Omitted from JSON sent to osqueryd. Mutually exclusive at
	// pack level with an enabled default_rrule_schedule (see ValidatePackScheduleDefaults). When set,
	// every query in the pack must use native scheduling after merge (ValidatePackQueriesAfterMerge).
	DefaultNativeSchedule NativeSchedule `config:"default_native_schedule" json:"-"`
	// DefaultRRuleSchedule provides RRULE defaults for queries that do not define rrule_schedule.
	// Config key default_rrule_schedule. Omitted from JSON sent to osqueryd.
	// When enabled, every query in the pack must use rrule_schedule after merge.
	DefaultRRuleSchedule *RRuleScheduleConfig `config:"default_rrule_schedule,omitempty" json:"-"`
	// DefaultSpaceID is applied to queries that omit space_id (native and RRULE).
	DefaultSpaceID string `config:"default_space_id,omitempty" json:"-"`

	Queries map[string]Query `config:"queries" json:"queries,omitempty"`
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
	ElasticOptions        *ElasticOptions        `config:"elastic_options" json:"-"`
	Schedule              map[string]Query       `config:"schedule" json:"schedule,omitempty"`
	Packs                 map[string]Pack        `config:"packs" json:"packs,omitempty"`
	Filepaths             map[string][]string    `config:"file_paths" json:"file_paths,omitempty"`
	Views                 map[string]string      `config:"views" json:"views,omitempty"`
	Events                *Events                `config:"events" json:"events,omitempty"`
	Yara                  map[string]interface{} `config:"yara" json:"yara,omitempty"`
	PrometheusTargets     map[string]interface{} `config:"prometheus_targets" json:"prometheus_targets,omitempty"`
	AutoTableConstruction map[string]interface{} `config:"auto_table_construction" json:"auto_table_construction,omitempty"`
}

// forOsqueryd returns a copy of c without queries that osquerybeat runs via RRULE (they would
// otherwise appear with interval 0 and are not meant for osqueryd's native scheduler).
func (c OsqueryConfig) forOsqueryd() OsqueryConfig {
	out := c
	out.Schedule = nil
	if len(c.Schedule) > 0 {
		for name, q := range c.Schedule {
			if q.RRuleSchedule.IsEnabled() {
				continue
			}
			if out.Schedule == nil {
				out.Schedule = make(map[string]Query)
			}
			out.Schedule[name] = q
		}
	}
	out.Packs = nil
	if len(c.Packs) > 0 {
		for packName, pack := range c.Packs {
			np := pack
			np.Queries = nil
			for qname, q := range pack.Queries {
				if q.RRuleSchedule.IsEnabled() {
					continue
				}
				if np.Queries == nil {
					np.Queries = make(map[string]Query)
				}
				np.Queries[qname] = q
			}
			if len(np.Queries) == 0 {
				continue
			}
			if out.Packs == nil {
				out.Packs = make(map[string]Pack)
			}
			out.Packs[packName] = np
		}
	}
	return out
}

// Render serializes the OsqueryConfig to JSON for osqueryd configuration.
func (c OsqueryConfig) Render() ([]byte, error) {
	return json.MarshalIndent(c.forOsqueryd(), "", "    ")
}
