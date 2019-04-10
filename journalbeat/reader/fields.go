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

//+build linux,cgo

package reader

import "github.com/coreos/go-systemd/sdjournal"

type fieldConversion struct {
	name      string
	isInteger bool
	dropped   bool
}

var (
	journaldEventFields = map[string]fieldConversion{
		// provided by systemd journal
		"COREDUMP_UNIT":                              fieldConversion{"journald.coredump.unit", false, false},
		"COREDUMP_USER_UNIT":                         fieldConversion{"journald.coredump.user_unit", false, false},
		"OBJECT_AUDIT_LOGINUID":                      fieldConversion{"journald.object.audit.login_uid", true, false},
		"OBJECT_AUDIT_SESSION":                       fieldConversion{"journald.object.audit.session", true, false},
		"OBJECT_CMDLINE":                             fieldConversion{"journald.object.cmd", false, false},
		"OBJECT_COMM":                                fieldConversion{"journald.object.name", false, false},
		"OBJECT_EXE":                                 fieldConversion{"journald.object.executable", false, false},
		"OBJECT_GID":                                 fieldConversion{"journald.object.gid", true, false},
		"OBJECT_PID":                                 fieldConversion{"journald.object.pid", true, false},
		"OBJECT_SYSTEMD_OWNER_UID":                   fieldConversion{"journald.object.systemd.owner_uid", true, false},
		"OBJECT_SYSTEMD_SESSION":                     fieldConversion{"journald.object.systemd.session", false, false},
		"OBJECT_SYSTEMD_UNIT":                        fieldConversion{"journald.object.systemd.unit", false, false},
		"OBJECT_SYSTEMD_USER_UNIT":                   fieldConversion{"journald.object.systemd.user_unit", false, false},
		"OBJECT_UID":                                 fieldConversion{"journald.object.uid", true, false},
		"_KERNEL_DEVICE":                             fieldConversion{"journald.kernel.device", false, false},
		"_KERNEL_SUBSYSTEM":                          fieldConversion{"journald.kernel.subsystem", false, false},
		"_SYSTEMD_INVOCATION_ID":                     fieldConversion{"systemd.invocation_id", false, false},
		"_SYSTEMD_USER_SLICE":                        fieldConversion{"systemd.user_slice", false, false},
		"_UDEV_DEVLINK":                              fieldConversion{"journald.kernel.device_symlinks", false, false}, // TODO aggregate multiple elements
		"_UDEV_DEVNODE":                              fieldConversion{"journald.kernel.device_node_path", false, false},
		"_UDEV_SYSNAME":                              fieldConversion{"journald.kernel.device_name", false, false},
		sdjournal.SD_JOURNAL_FIELD_AUDIT_LOGINUID:    fieldConversion{"process.audit.login_uid", true, false},
		sdjournal.SD_JOURNAL_FIELD_AUDIT_SESSION:     fieldConversion{"process.audit.session", false, false},
		sdjournal.SD_JOURNAL_FIELD_BOOT_ID:           fieldConversion{"host.boot_id", false, false},
		sdjournal.SD_JOURNAL_FIELD_CAP_EFFECTIVE:     fieldConversion{"process.capabilites", false, false},
		sdjournal.SD_JOURNAL_FIELD_CMDLINE:           fieldConversion{"process.cmd", false, false},
		sdjournal.SD_JOURNAL_FIELD_CODE_FILE:         fieldConversion{"journald.code.file", false, false},
		sdjournal.SD_JOURNAL_FIELD_CODE_FUNC:         fieldConversion{"journald.code.func", false, false},
		sdjournal.SD_JOURNAL_FIELD_CODE_LINE:         fieldConversion{"journald.code.line", true, false},
		sdjournal.SD_JOURNAL_FIELD_COMM:              fieldConversion{"process.name", false, false},
		sdjournal.SD_JOURNAL_FIELD_EXE:               fieldConversion{"process.executable", false, false},
		sdjournal.SD_JOURNAL_FIELD_GID:               fieldConversion{"process.uid", true, false},
		sdjournal.SD_JOURNAL_FIELD_HOSTNAME:          fieldConversion{"host.hostname", false, false},
		sdjournal.SD_JOURNAL_FIELD_MACHINE_ID:        fieldConversion{"host.id", false, false},
		sdjournal.SD_JOURNAL_FIELD_MESSAGE:           fieldConversion{"message", false, false},
		sdjournal.SD_JOURNAL_FIELD_PID:               fieldConversion{"process.pid", true, false},
		sdjournal.SD_JOURNAL_FIELD_PRIORITY:          fieldConversion{"syslog.priority", true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_FACILITY:   fieldConversion{"syslog.facility", true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER: fieldConversion{"syslog.identifier", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_PID:        fieldConversion{"syslog.pid", true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_CGROUP:    fieldConversion{"systemd.cgroup", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_OWNER_UID: fieldConversion{"systemd.owner_uid", true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SESSION:   fieldConversion{"systemd.session", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SLICE:     fieldConversion{"systemd.slice", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT:      fieldConversion{"systemd.unit", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_USER_UNIT: fieldConversion{"systemd.user_unit", false, false},
		sdjournal.SD_JOURNAL_FIELD_TRANSPORT:         fieldConversion{"systemd.transport", false, false},
		sdjournal.SD_JOURNAL_FIELD_UID:               fieldConversion{"process.uid", true, false},

		// docker journald fields from: https://docs.docker.com/config/containers/logging/journald/
		"CONTAINER_ID":              fieldConversion{"container.id_truncated", false, false},
		"CONTAINER_ID_FULL":         fieldConversion{"container.id", false, false},
		"CONTAINER_NAME":            fieldConversion{"container.name", false, false},
		"CONTAINER_TAG":             fieldConversion{"container.log.tag", false, false},
		"CONTAINER_PARTIAL_MESSAGE": fieldConversion{"container.partial", false, false},

		// dropped fields
		sdjournal.SD_JOURNAL_FIELD_MONOTONIC_TIMESTAMP:       fieldConversion{"", false, true}, // saved in the registry
		sdjournal.SD_JOURNAL_FIELD_SOURCE_REALTIME_TIMESTAMP: fieldConversion{"", false, true}, // saved in the registry
		sdjournal.SD_JOURNAL_FIELD_CURSOR:                    fieldConversion{"", false, true}, // saved in the registry
		"_SOURCE_MONOTONIC_TIMESTAMP":                        fieldConversion{"", false, true}, // received timestamp stored in @timestamp
	}
)
