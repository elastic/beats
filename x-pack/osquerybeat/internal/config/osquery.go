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

	// Always enforced as snapshot, can't be changed via configuration
	Snapshot bool `json:"snapshot"`
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

func (c OsqueryConfig) Render() ([]byte, error) {
	return json.MarshalIndent(c, "", "    ")
}
