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

// Details about the event's logging mechanism or logging transport.
// The log.* fields are typically populated with details about the logging
// mechanism used to create and/or transport the event. For example, syslog
// details belong under `log.syslog.*`.
// The details specific to your event source are typically not logged under
// `log.*`, but rather in `event.*` or in other ECS fields.
type Log struct {
	// Original log level of the log event.
	// If the source of the event provides a log level or textual severity,
	// this is the one that goes in `log.level`. If your source doesn't specify
	// one, you may put your event transport's severity here (e.g. Syslog
	// severity).
	// Some examples are `warn`, `err`, `i`, `informational`.
	Level string `ecs:"level"`

	// Full path to the log file this event came from, including the file name.
	// It should include the drive letter, when appropriate.
	// If the event wasn't read from a log file, do not populate this field.
	FilePath string `ecs:"file.path"`

	// The name of the logger inside an application. This is usually the name
	// of the class which initialized the logger, or can be a custom name.
	Logger string `ecs:"logger"`

	// The name of the file containing the source code which originated the log
	// event.
	// Note that this field is not meant to capture the log file. The correct
	// field to capture the log file is `log.file.path`.
	OriginFileName string `ecs:"origin.file.name"`

	// The line number of the file containing the source code which originated
	// the log event.
	OriginFileLine int64 `ecs:"origin.file.line"`

	// The name of the function or method which originated the log event.
	OriginFunction string `ecs:"origin.function"`

	// The Syslog metadata of the event, if the event was transmitted via
	// Syslog. Please see RFCs 5424 or 3164.
	Syslog map[string]interface{} `ecs:"syslog"`

	// The Syslog numeric severity of the log event, if available.
	// If the event source publishing via Syslog provides a different numeric
	// severity value (e.g. firewall, IDS), your source's numeric severity
	// should go to `event.severity`. If the event source does not specify a
	// distinct severity, you can optionally copy the Syslog severity to
	// `event.severity`.
	SyslogSeverityCode int64 `ecs:"syslog.severity.code"`

	// The Syslog numeric severity of the log event, if available.
	// If the event source publishing via Syslog provides a different severity
	// value (e.g. firewall, IDS), your source's text severity should go to
	// `log.level`. If the event source does not specify a distinct severity,
	// you can optionally copy the Syslog severity to `log.level`.
	SyslogSeverityName string `ecs:"syslog.severity.name"`

	// The Syslog numeric facility of the log event, if available.
	// According to RFCs 5424 and 3164, this value should be an integer between
	// 0 and 23.
	SyslogFacilityCode int64 `ecs:"syslog.facility.code"`

	// The Syslog text-based facility of the log event, if available.
	SyslogFacilityName string `ecs:"syslog.facility.name"`

	// Syslog numeric priority of the event, if available.
	// According to RFCs 5424 and 3164, the priority is 8 * facility +
	// severity. This number is therefore expected to contain a value between 0
	// and 191.
	SyslogPriority int64 `ecs:"syslog.priority"`

	// The device or application that originated the Syslog message, if available.
	SyslogAppname string `ecs:"syslog.appname"`

	// The hostname, FQDN, or IP of the machine that originally sent the
	// Syslog message. This is sourced from the hostname field of the syslog header.
	// Depending on the environment, this value may be different from the host that
	// handled the event, especially if the host handling the events is acting as
	// a collector.
	SyslogHostname string `ecs:"syslog.hostname"`

	// An identifier for the type of Syslog message, if available. Only
	// applicable for RFC 5424 messages.
	SyslogMsgid string `ecs:"syslog.msgid"`

	// The process name or ID that originated the Syslog message, if available.
	SyslogProcid string `ecs:"syslog.procid"`

	// Structured data expressed in RFC 5424 messages, if available. These
	// are key-value pairs formed from the structured data portion of the syslog
	// message, as defined in RFC 5424 Section 6.3.
	SyslogStructured_data map[string]interface{} `ecs:"syslog.syslog.structured_data"`

	// The version of the Syslog protocol specification. Only applicable
	// for RFC 5424 messages.
	SyslogVersion string `ecs:"syslog.version"`
}
