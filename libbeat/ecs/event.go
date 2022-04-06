// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ecs

import (
	"time"
)

// The event fields are used for context information about the log or metric
// event itself.
// A log is defined as an event containing details of something that happened.
// Log events must include the time at which the thing happened. Examples of
// log events include a process starting on a host, a network packet being sent
// from a source to a destination, or a network connection between a client and
// a server being initiated or closed. A metric is defined as an event
// containing one or more numerical measurements and the time at which the
// measurement was taken. Examples of metric events include memory pressure
// measured on a host and device temperature. See the `event.kind` definition
// in this section for additional details about metric and state events.
type Event struct {
	// Unique ID to describe the event.
	ID string `ecs:"id"`

	// Identification code for this event, if one exists.
	// Some event sources use event codes to identify messages unambiguously,
	// regardless of message language or wording adjustments over time. An
	// example of this is the Windows Event ID.
	Code string `ecs:"code"`

	// This is one of four ECS Categorization Fields, and indicates the highest
	// level in the ECS category hierarchy.
	// `event.kind` gives high-level information about what type of information
	// the event contains, without being specific to the contents of the event.
	// For example, values of this field distinguish alert events from metric
	// events.
	// The value of this field can be used to inform how these kinds of events
	// should be handled. They may warrant different retention, different
	// access control, it may also help understand whether the data coming in
	// at a regular interval or not.
	Kind string `ecs:"kind"`

	// This is one of four ECS Categorization Fields, and indicates the second
	// level in the ECS category hierarchy.
	// `event.category` represents the "big buckets" of ECS categories. For
	// example, filtering on `event.category:process` yields all events
	// relating to process activity. This field is closely related to
	// `event.type`, which is used as a subcategory.
	// This field is an array. This will allow proper categorization of some
	// events that fall in multiple categories.
	Category []string `ecs:"category"`

	// The action captured by the event.
	// This describes the information in the event. It is more specific than
	// `event.category`. Examples are `group-add`, `process-started`,
	// `file-created`. The value is normally defined by the implementer.
	Action string `ecs:"action"`

	// This is one of four ECS Categorization Fields, and indicates the lowest
	// level in the ECS category hierarchy.
	// `event.outcome` simply denotes whether the event represents a success or
	// a failure from the perspective of the entity that produced the event.
	// Note that when a single transaction is described in multiple events,
	// each event may populate different values of `event.outcome`, according
	// to their perspective.
	// Also note that in the case of a compound event (a single event that
	// contains multiple logical events), this field should be populated with
	// the value that best captures the overall success or failure from the
	// perspective of the event producer.
	// Further note that not all events will have an associated outcome. For
	// example, this field is generally not populated for metric events, events
	// with `event.type:info`, or any events for which an outcome does not make
	// logical sense.
	Outcome string `ecs:"outcome"`

	// This is one of four ECS Categorization Fields, and indicates the third
	// level in the ECS category hierarchy.
	// `event.type` represents a categorization "sub-bucket" that, when used
	// along with the `event.category` field values, enables filtering events
	// down to a level appropriate for single visualization.
	// This field is an array. This will allow proper categorization of some
	// events that fall in multiple event types.
	Type []string `ecs:"type"`

	// Name of the module this data is coming from.
	// If your monitoring agent supports the concept of modules or plugins to
	// process events of a given source (e.g. Apache logs), `event.module`
	// should contain the name of this module.
	Module string `ecs:"module"`

	// Name of the dataset.
	// If an event source publishes more than one type of log or events (e.g.
	// access log, error log), the dataset is used to specify which one the
	// event comes from.
	// It's recommended but not required to start the dataset name with the
	// module name, followed by a dot, then the dataset name.
	Dataset string `ecs:"dataset"`

	// Source of the event.
	// Event transports such as Syslog or the Windows Event Log typically
	// mention the source of an event. It can be the name of the software that
	// generated the event (e.g. Sysmon, httpd), or of a subsystem of the
	// operating system (kernel, Microsoft-Windows-Security-Auditing).
	Provider string `ecs:"provider"`

	// The numeric severity of the event according to your event source.
	// What the different severity values mean can be different between sources
	// and use cases. It's up to the implementer to make sure severities are
	// consistent across events from the same source.
	// The Syslog severity belongs in `log.syslog.severity.code`.
	// `event.severity` is meant to represent the severity according to the
	// event source (e.g. firewall, IDS). If the event source does not publish
	// its own severity, you may optionally copy the `log.syslog.severity.code`
	// to `event.severity`.
	Severity int64 `ecs:"severity"`

	// Raw text message of entire event. Used to demonstrate log integrity or
	// where the full log message (before splitting it up in multiple parts)
	// may be required, e.g. for reindex.
	// This field is not indexed and doc_values are disabled. It cannot be
	// searched, but it can be retrieved from `_source`. If users wish to
	// override this and index this field, please see `Field data types` in the
	// `Elasticsearch Reference`.
	Original string `ecs:"original"`

	// Hash (perhaps logstash fingerprint) of raw field to be able to
	// demonstrate log integrity.
	Hash string `ecs:"hash"`

	// Duration of the event in nanoseconds.
	// If event.start and event.end are known this value should be the
	// difference between the end and start time.
	Duration time.Duration `ecs:"duration"`

	// Sequence number of the event.
	// The sequence number is a value published by some event sources, to make
	// the exact ordering of events unambiguous, regardless of the timestamp
	// precision.
	Sequence int64 `ecs:"sequence"`

	// This field should be populated when the event's timestamp does not
	// include timezone information already (e.g. default Syslog timestamps).
	// It's optional otherwise.
	// Acceptable timezone formats are: a canonical ID (e.g.
	// "Europe/Amsterdam"), abbreviated (e.g. "EST") or an HH:mm differential
	// (e.g. "-05:00").
	Timezone string `ecs:"timezone"`

	// event.created contains the date/time when the event was first read by an
	// agent, or by your pipeline.
	// This field is distinct from @timestamp in that @timestamp typically
	// contain the time extracted from the original event.
	// In most situations, these two timestamps will be slightly different. The
	// difference can be used to calculate the delay between your source
	// generating an event, and the time when your agent first processed it.
	// This can be used to monitor your agent's or pipeline's ability to keep
	// up with your event source.
	// In case the two timestamps are identical, @timestamp should be used.
	Created time.Time `ecs:"created"`

	// event.start contains the date when the event started or when the
	// activity was first observed.
	Start time.Time `ecs:"start"`

	// event.end contains the date when the event ended or when the activity
	// was last observed.
	End time.Time `ecs:"end"`

	// Risk score or priority of the event (e.g. security solutions). Use your
	// system's original value here.
	RiskScore float64 `ecs:"risk_score"`

	// Normalized risk score or priority of the event, on a scale of 0 to 100.
	// This is mainly useful if you use more than one system that assigns risk
	// scores, and you want to see a normalized value across all systems.
	RiskScoreNorm float64 `ecs:"risk_score_norm"`

	// Timestamp when an event arrived in the central data store.
	// This is different from `@timestamp`, which is when the event originally
	// occurred.  It's also different from `event.created`, which is meant to
	// capture the first time an agent saw the event.
	// In normal conditions, assuming no tampering, the timestamps should
	// chronologically look like this: `@timestamp` < `event.created` <
	// `event.ingested`.
	Ingested time.Time `ecs:"ingested"`

	// Reference URL linking to additional information about this event.
	// This URL links to a static definition of this event. Alert events,
	// indicated by `event.kind:alert`, are a common use case for this field.
	Reference string `ecs:"reference"`

	// URL linking to an external system to continue investigation of this
	// event.
	// This URL links to another system where in-depth investigation of the
	// specific occurrence of this event can take place. Alert events,
	// indicated by `event.kind:alert`, are a common use case for this field.
	Url string `ecs:"url"`

	// Reason why this event happened, according to the source.
	// This describes the why of a particular action or outcome captured in the
	// event. Where `event.action` captures the action from the event,
	// `event.reason` describes why that action was taken. For example, a web
	// proxy with an `event.action` which denied the request may also populate
	// `event.reason` with the reason why (e.g. `blocked site`).
	Reason string `ecs:"reason"`

	// Agents are normally responsible for populating the `agent.id` field
	// value. If the system receiving events is capable of validating the value
	// based on authentication information for the client then this field can
	// be used to reflect the outcome of that validation.
	// For example if the agent's connection is authenticated with mTLS and the
	// client cert contains the ID of the agent to which the cert was issued
	// then the `agent.id` value in events can be checked against the
	// certificate. If the values match then `event.agent_id_status: verified`
	// is added to the event, otherwise one of the other allowed values should
	// be used.
	// If no validation is performed then the field should be omitted.
	// The allowed values are:
	// `verified` - The `agent.id` field value matches expected value obtained
	// from auth metadata.
	// `mismatch` - The `agent.id` field value does not match the expected
	// value obtained from auth metadata.
	// `missing` - There was no `agent.id` field in the event to validate.
	// `auth_metadata_missing` - There was no auth metadata or it was missing
	// information about the agent ID.
	AgentIDStatus string `ecs:"agent_id_status"`
}
