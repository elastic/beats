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

//go:build linux && cgo

package journalfield

// journaldEventFields provides default field mappings and conversions rules.
var journaldEventFields = FieldConversion{
	// provided by systemd journal
	"COREDUMP_UNIT":            text("journald.coredump.unit"),
	"COREDUMP_USER_UNIT":       text("journald.coredump.user_unit"),
	"MESSAGE":                  text("message"),
	"MESSAGE_ID":               text("message_id"),
	"OBJECT_AUDIT_LOGINUID":    integer("journald.object.audit.login_uid"),
	"OBJECT_AUDIT_SESSION":     integer("journald.object.audit.session"),
	"OBJECT_CMDLINE":           text("journald.object.process.command_line"),
	"OBJECT_COMM":              text("journald.object.process.name"),
	"OBJECT_EXE":               text("journald.object.process.executable"),
	"OBJECT_GID":               integer("journald.object.gid"),
	"OBJECT_PID":               integer("journald.object.pid"),
	"OBJECT_SYSTEMD_OWNER_UID": integer("journald.object.systemd.owner_uid"),
	"OBJECT_SYSTEMD_SESSION":   text("journald.object.systemd.session"),
	"OBJECT_SYSTEMD_UNIT":      text("journald.object.systemd.unit"),
	"OBJECT_SYSTEMD_USER_UNIT": text("journald.object.systemd.user_unit"),
	"OBJECT_UID":               integer("journald.object.uid"),
	"PRIORITY":                 integer("syslog.priority", "log.syslog.priority"),
	"SYSLOG_FACILITY":          integer("syslog.facility", "log.syslog.facility.code"),
	"SYSLOG_IDENTIFIER":        text("syslog.identifier"),
	"SYSLOG_PID":               integer("syslog.pid"),
	"UNIT":                     text("journald.unit"),
	"_AUDIT_LOGINUID":          integer("journald.audit.login_uid"),
	"_AUDIT_SESSION":           text("journald.audit.session"),
	"_BOOT_ID":                 text("journald.host.boot_id"),
	"_CAP_EFFECTIVE":           text("journald.process.capabilities"),
	"_CMDLINE":                 text("journald.process.command_line"),
	"CODE_FILE":                text("journald.code.file"),
	"CODE_FUNC":                text("journald.code.func"),
	"CODE_LINE":                integer("journald.code.line"),
	"_COMM":                    text("journald.process.name"),
	"_EXE":                     text("journald.process.executable"),
	"_GID":                     integer("journald.gid"),
	"_HOSTNAME":                text("host.hostname"),
	"_KERNEL_DEVICE":           text("journald.kernel.device"),
	"_KERNEL_SUBSYSTEM":        text("journald.kernel.subsystem"),
	"_MACHINE_ID":              text("host.id"),
	"_PID":                     integer("journald.pid"),
	"_SYSTEMD_CGROUP":          text("systemd.cgroup"),
	"_SYSTEMD_INVOCATION_ID":   text("systemd.invocation_id"),
	"_SYSTEMD_OWNER_UID":       integer("systemd.owner_uid"),
	"_SYSTEMD_SESSION":         text("systemd.session"),
	"_SYSTEMD_SLICE":           text("systemd.slice"),
	"_SYSTEMD_UNIT":            text("systemd.unit"),
	"_SYSTEMD_USER_SLICE":      text("systemd.user_slice"),
	"_SYSTEMD_USER_UNIT":       text("systemd.user_unit"),
	"_TRANSPORT":               text("systemd.transport"),
	"_UDEV_DEVLINK":            text("journald.kernel.device_symlinks"),
	"_UDEV_DEVNODE":            text("journald.kernel.device_node_path"),
	"_UDEV_SYSNAME":            text("journald.kernel.device_name"),
	"_UID":                     integer("journald.uid"),

	// docker journald fields from: https://docs.docker.com/config/containers/logging/journald/
	"CONTAINER_ID":              text("container.id_truncated"),
	"CONTAINER_ID_FULL":         text("container.id"),
	"CONTAINER_NAME":            text("container.name"),
	"CONTAINER_TAG":             text("container.log.tag"),
	"CONTAINER_PARTIAL_MESSAGE": text("container.partial"),

	// dropped fields
	"_SOURCE_MONOTONIC_TIMESTAMP": ignoredField, // received timestamp stored in @timestamp
	"_SOURCE_REALTIME_TIMESTAMP":  ignoredField, // saved in the registry
	"__CURSOR":                    ignoredField, // saved in the registry
	"__MONOTONIC_TIMESTAMP":       ignoredField, // saved in the registry
}
