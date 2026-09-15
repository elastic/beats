// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"encoding/json"
)

<<<<<<< HEAD
// ElasticOptions contains Beat-specific options that are not part of
// osquery's native config schema.
type ElasticOptions struct {
	Install             *InstallConfig             `config:"install" json:"-"`
	QueryProfileStorage *QueryProfileStorageConfig `config:"query_profile_storage" json:"-"`
=======
const (
	// MaxSplay is the maximum allowed splay duration (aligns with daily-or-longer RRULE minimum).
	MaxSplay = 12 * time.Hour
	// DefaultSplay is the default splay duration (disabled)
	DefaultSplay = 0
	// DefaultCheckTimeout is the osqueryd --version startup check deadline
	// when elastic_options.check_timeout is unset.
	DefaultCheckTimeout = 15 * time.Second
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
	t = t.UTC()
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
	t = t.UTC()
	return &t, nil
}

// IsEnabled returns true if an RRULE is configured
func (c *RRuleScheduleConfig) IsEnabled() bool {
	return c != nil && c.RRule != ""
}

// ElasticOptions contains Beat-specific options that are not part of
// osquery's native config schema.
type ElasticOptions struct {
	Install *InstallConfig `config:"install" json:"-"`
	// Profiling groups all query profiling settings (global publish default and local storage).
	Profiling *ProfilingConfig `config:"profiling" json:"-"`
	// Extensions configures loading of customer-managed osquery extensions.
	Extensions *ExtensionsConfig `config:"extensions" json:"-"`
	// CheckTimeout optionally overrides the osqueryd --version startup check
	// deadline. Go duration format ("15s", "30s", "1m"). Default: 15s.
	CheckTimeout string `config:"check_timeout" json:"-"`
}

// ParseCheckTimeout parses elastic_options.check_timeout. An empty value
// returns DefaultCheckTimeout. The duration must be greater than zero.
func ParseCheckTimeout(raw string) (time.Duration, error) {
	if raw == "" {
		return DefaultCheckTimeout, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid osquery.elastic_options.check_timeout %q: %w", raw, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("osquery.elastic_options.check_timeout must be greater than 0, got %s", raw)
	}
	return d, nil
}

// ExtensionsConfig configures loading of customer-managed (third-party or
// customer-built) osquery extensions. These extensions are NOT developed,
// validated, or supported by Elastic; customers are fully responsible for
// their security, maintenance, and stability.
//
// Paths lists absolute entries on the endpoint, where each entry may be:
//   - a directory: every extension binary found directly in it is autoloaded
//     (files ending in ".ext" on Unix or ".exe" on Windows);
//   - a file: that specific extension binary is autoloaded;
//   - a glob pattern (containing *, ? or [ ]): each match is resolved as a
//     directory or file per the rules above.
//
// Symlinks are rejected (entries, glob matches, and directory contents) so the
// binary that is validated is the one osqueryd executes.
//
// Entries are resolved when osqueryd is (re)started, which happens when the
// extension configuration (or other osquery options) change. Adding or removing
// binaries in a configured directory does NOT trigger a reload by itself.
//
// osquerybeat never copies these binaries and never writes into the Elastic
// Agent install tree; it only appends the resolved paths to the osquery
// extensions autoload file that lives in the runtime data directory. osqueryd
// enforces safe file permissions on autoloaded extensions (owned by the running
// user, not writable by group or others); unsafe, non-executable, or otherwise
// invalid binaries are skipped and logged rather than aborting startup.
type ExtensionsConfig struct {
	// Paths lists absolute directories, files, or glob patterns to resolve into
	// extension binaries.
	Paths []string `config:"paths" json:"-"`
	// Timeout optionally overrides osquery's extensions_timeout (seconds), the
	// time osqueryd waits for autoloaded extensions to register.
	Timeout int `config:"timeout" json:"-"`
	// Require lists extension names osqueryd must wait for at startup
	// (osquery's extensions_require); queries do not run until the named
	// extensions have registered or extensions_timeout elapses. Use this to
	// avoid "no such table" races against slow-registering extensions.
	Require []string `config:"require" json:"-"`
}

// PathsOrEmpty returns the configured extension paths, or an empty slice when no
// extensions are configured.
func (c *ExtensionsConfig) PathsOrEmpty() []string {
	if c == nil {
		return nil
	}
	return c.Paths
}

// ProfilingConfig groups all query profiling settings. When ProfilingAll is enabled (the default),
// profiles are collected and published to the osquery_manager.query_profile data stream for all
// queries that do not set their own profiling flag. It still requires the query_profile input stream
// to be present for events to be published. Storage controls local retention of live profiles
// for diagnostics, independent of publishing.
type ProfilingConfig struct {
	ProfilingAll *bool                      `config:"profiling_all" json:"-"`
	Storage      *QueryProfileStorageConfig `config:"storage" json:"-"`
}

// ProfilingAllOrDefault returns the global profiling default, which is enabled unless
// explicitly set to false.
func (c ProfilingConfig) ProfilingAllOrDefault() bool {
	if c.ProfilingAll == nil {
		return true
	}
	return *c.ProfilingAll
}

// ResolveProfiling returns the effective profiling decision for a query: a per-query
// override wins when set, otherwise the global default applies.
func ResolveProfiling(globalDefault bool, override *bool) bool {
	if override != nil {
		return *override
	}
	return globalDefault
>>>>>>> 10b9af2 ([osquerybeat] Bound configurable startup check and harden clock-skew handling (#52992))
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

type Query struct {
	Query string `config:"query" json:"query"`
	// Native schedule fields.
	Interval int `config:"interval" json:"interval"`
	// ScheduleID is the policy-defined schedule identifier for this scheduled query.
	// Stored in the policy and used in result/response documents for correlation.
	// If empty, the query name is used as the schedule_id when publishing.
	ScheduleID string `config:"schedule_id,omitempty" json:"schedule_id,omitempty"`
	// StartDate is the optional start date for native (interval-based) schedules (RFC3339).
	// Used as the reference for schedule_execution_count.
	StartDate string `config:"start_date,omitempty" json:"start_date,omitempty"`
	// SpaceID is the optional policy space identifier for this scheduled query.
	SpaceID string `config:"space_id,omitempty" json:"space_id,omitempty"`

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

	// Optional internal flag to emit per-query profiling for this scheduled query.
	// This is consumed by osquerybeat and not rendered into osqueryd configuration.
	Profile bool `config:"profile" json:"-"`
}

type Pack struct {
	// PackID is the policy-defined pack identifier; used in result/response documents for correlation.
	// If empty, the pack map key (pack name) is used when publishing.
	PackID string `config:"pack_id,omitempty" json:"pack_id,omitempty"`
	// PackName is the policy-defined human-readable pack name; emitted as pack_name in
	// result/response documents alongside pack_id. Optional and not sent to osqueryd.
	PackName  string           `config:"pack_name,omitempty" json:"-"`
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

// Render serializes the OsqueryConfig to JSON for osqueryd configuration.
func (c OsqueryConfig) Render() ([]byte, error) {
	return json.MarshalIndent(c, "", "    ")
}
