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
// +build linux,cgo

package journalfield

import "github.com/coreos/go-systemd/v22/sdjournal"

// journaldEventFields provides default field mappings and conversions rules.
var journaldEventFields = FieldConversion{
	// provided by systemd journal
	"COREDUMP_UNIT":                              text("journald.coredump.unit"),
	"COREDUMP_USER_UNIT":                         text("journald.coredump.user_unit"),
	"OBJECT_AUDIT_LOGINUID":                      integer("journald.object.audit.login_uid"),
	"OBJECT_AUDIT_SESSION":                       integer("journald.object.audit.session"),
	"OBJECT_CMDLINE":                             text("journald.object.process.command_line"),
	"OBJECT_COMM":                                text("journald.object.process.name"),
	"OBJECT_EXE":                                 text("journald.object.process.executable"),
	"OBJECT_GID":                                 integer("journald.object.gid"),
	"OBJECT_PID":                                 integer("journald.object.pid"),
	"OBJECT_SYSTEMD_OWNER_UID":                   integer("journald.object.systemd.owner_uid"),
	"OBJECT_SYSTEMD_SESSION":                     text("journald.object.systemd.session"),
	"OBJECT_SYSTEMD_UNIT":                        text("journald.object.systemd.unit"),
	"OBJECT_SYSTEMD_USER_UNIT":                   text("journald.object.systemd.user_unit"),
	"OBJECT_UID":                                 integer("journald.object.uid"),
	"_KERNEL_DEVICE":                             text("journald.kernel.device"),
	"_KERNEL_SUBSYSTEM":                          text("journald.kernel.subsystem"),
	"_SYSTEMD_INVOCATION_ID":                     text("systemd.invocation_id"),
	"_SYSTEMD_USER_SLICE":                        text("systemd.user_slice"),
	"_UDEV_DEVLINK":                              text("journald.kernel.device_symlinks"),
	"_UDEV_DEVNODE":                              text("journald.kernel.device_node_path"),
	"_UDEV_SYSNAME":                              text("journald.kernel.device_name"),
	sdjournal.SD_JOURNAL_FIELD_AUDIT_LOGINUID:    integer("journald.audit.login_uid"),
	sdjournal.SD_JOURNAL_FIELD_AUDIT_SESSION:     text("journald.audit.session"),
	sdjournal.SD_JOURNAL_FIELD_BOOT_ID:           text("journald.host.boot_id"),
	sdjournal.SD_JOURNAL_FIELD_CAP_EFFECTIVE:     text("journald.process.capabilities"),
	sdjournal.SD_JOURNAL_FIELD_CMDLINE:           text("journald.process.command_line"),
	sdjournal.SD_JOURNAL_FIELD_CODE_FILE:         text("journald.code.file"),
	sdjournal.SD_JOURNAL_FIELD_CODE_FUNC:         text("journald.code.func"),
	sdjournal.SD_JOURNAL_FIELD_CODE_LINE:         integer("journald.code.line"),
	sdjournal.SD_JOURNAL_FIELD_COMM:              text("journald.process.name"),
	sdjournal.SD_JOURNAL_FIELD_EXE:               text("journald.process.executable"),
	sdjournal.SD_JOURNAL_FIELD_GID:               integer("journald.gid"),
	sdjournal.SD_JOURNAL_FIELD_HOSTNAME:          text("host.hostname"),
	sdjournal.SD_JOURNAL_FIELD_MACHINE_ID:        text("host.id"),
	sdjournal.SD_JOURNAL_FIELD_MESSAGE:           text("message"),
	sdjournal.SD_JOURNAL_FIELD_PID:               integer("journald.pid"),
	sdjournal.SD_JOURNAL_FIELD_PRIORITY:          integer("syslog.priority", "log.syslog.priority"),
	sdjournal.SD_JOURNAL_FIELD_SYSLOG_FACILITY:   integer("syslog.facility", "log.syslog.facility.code"),
	sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER: text("syslog.identifier"),
	sdjournal.SD_JOURNAL_FIELD_SYSLOG_PID:        integer("syslog.pid"),
	sdjournal.SD_JOURNAL_FIELD_SYSTEMD_CGROUP:    text("systemd.cgroup"),
	sdjournal.SD_JOURNAL_FIELD_SYSTEMD_OWNER_UID: integer("systemd.owner_uid"),
	sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SESSION:   text("systemd.session"),
	sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SLICE:     text("systemd.slice"),
	sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT:      text("systemd.unit"),
	sdjournal.SD_JOURNAL_FIELD_SYSTEMD_USER_UNIT: text("systemd.user_unit"),
	sdjournal.SD_JOURNAL_FIELD_TRANSPORT:         text("systemd.transport"),
	sdjournal.SD_JOURNAL_FIELD_UID:               integer("journald.uid"),

	// docker journald fields from: https://docs.docker.com/config/containers/logging/journald/
	"CONTAINER_ID":              text("container.id_truncated"),
	"CONTAINER_ID_FULL":         text("container.id"),
	"CONTAINER_NAME":            text("container.name"),
	"CONTAINER_TAG":             text("container.log.tag"),
	"CONTAINER_PARTIAL_MESSAGE": text("container.partial"),

	// dropped fields
	sdjournal.SD_JOURNAL_FIELD_MONOTONIC_TIMESTAMP:       ignoredField, // saved in the registry
	sdjournal.SD_JOURNAL_FIELD_SOURCE_REALTIME_TIMESTAMP: ignoredField, // saved in the registry
	sdjournal.SD_JOURNAL_FIELD_CURSOR:                    ignoredField, // saved in the registry
	"_SOURCE_MONOTONIC_TIMESTAMP":                        ignoredField, // received timestamp stored in @timestamp
}
