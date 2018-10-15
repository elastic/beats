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

package reader

import "github.com/coreos/go-systemd/sdjournal"

type FieldConversion struct {
	Name      string
	IsInteger bool
	Dropped   bool
}

var (
	journaldEventFields = map[string]FieldConversion{
		// provided by systemd journal
		"COREDUMP_UNIT":                              FieldConversion{"journald.coredump.unit", false, false},
		"COREDUMP_USER_UNIT":                         FieldConversion{"journald.coredump.user_unit", false, false},
		"OBJECT_AUDIT_LOGINUID":                      FieldConversion{"journald.object.audit.login_uid", true, false},
		"OBJECT_AUDIT_SESSION":                       FieldConversion{"journald.object.audit.session", true, false},
		"OBJECT_CMDLINE":                             FieldConversion{"journald.object.cmd", false, false},
		"OBJECT_COMM":                                FieldConversion{"journald.object.name", false, false},
		"OBJECT_EXE":                                 FieldConversion{"journald.object.executable", false, false},
		"OBJECT_GID":                                 FieldConversion{"journald.object.gid", true, false},
		"OBJECT_PID":                                 FieldConversion{"journald.object.pid", true, false},
		"OBJECT_SYSTEMD_OWNER_UID":                   FieldConversion{"journald.object.systemd.owner_uid", true, false},
		"OBJECT_SYSTEMD_SESSION":                     FieldConversion{"journald.object.systemd.session", false, false},
		"OBJECT_SYSTEMD_UNIT":                        FieldConversion{"journald.object.systemd.unit", false, false},
		"OBJECT_SYSTEMD_USER_UNIT":                   FieldConversion{"journald.object.systemd.user_unit", false, false},
		"OBJECT_UID":                                 FieldConversion{"journald.object.uid", true, false},
		"_KERNEL_DEVICE":                             FieldConversion{"journald.kernel.device", false, false},
		"_KERNEL_SUBSYSTEM":                          FieldConversion{"journald.kernel.subsystem", false, false},
		"_SYSTEMD_INVOCATION_ID":                     FieldConversion{"systemd.invocation_id", false, false},
		"_SYSTEMD_USER_SLICE":                        FieldConversion{"systemd.user_slice", false, false},
		"_UDEV_DEVLINK":                              FieldConversion{"journald.kernel.device_symlinks", false, false}, // TODO aggregate multiple elements
		"_UDEV_DEVNODE":                              FieldConversion{"journald.kernel.device_node_path", false, false},
		"_UDEV_SYSNAME":                              FieldConversion{"journald.kernel.device_name", false, false},
		sdjournal.SD_JOURNAL_FIELD_AUDIT_LOGINUID:    FieldConversion{"process.audit.login_uid", true, false},
		sdjournal.SD_JOURNAL_FIELD_AUDIT_SESSION:     FieldConversion{"process.audit.session", false, false},
		sdjournal.SD_JOURNAL_FIELD_BOOT_ID:           FieldConversion{"host.boot_id", false, false},
		sdjournal.SD_JOURNAL_FIELD_CAP_EFFECTIVE:     FieldConversion{"process.capabilites", false, false},
		sdjournal.SD_JOURNAL_FIELD_CMDLINE:           FieldConversion{"process.cmd", false, false},
		sdjournal.SD_JOURNAL_FIELD_CODE_FILE:         FieldConversion{"journald.code.file", false, false},
		sdjournal.SD_JOURNAL_FIELD_CODE_FUNC:         FieldConversion{"journald.code.func", false, false},
		sdjournal.SD_JOURNAL_FIELD_CODE_LINE:         FieldConversion{"journald.code.line", true, false},
		sdjournal.SD_JOURNAL_FIELD_COMM:              FieldConversion{"process.name", false, false},
		sdjournal.SD_JOURNAL_FIELD_EXE:               FieldConversion{"process.executable", false, false},
		sdjournal.SD_JOURNAL_FIELD_GID:               FieldConversion{"process.uid", true, false},
		sdjournal.SD_JOURNAL_FIELD_HOSTNAME:          FieldConversion{"host.name", false, false},
		sdjournal.SD_JOURNAL_FIELD_MACHINE_ID:        FieldConversion{"host.id", true, false},
		sdjournal.SD_JOURNAL_FIELD_MESSAGE:           FieldConversion{"message", false, false},
		sdjournal.SD_JOURNAL_FIELD_PID:               FieldConversion{"process.pid", true, false},
		sdjournal.SD_JOURNAL_FIELD_PRIORITY:          FieldConversion{"syslog.priority", true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_FACILITY:   FieldConversion{"syslog.facility", true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER: FieldConversion{"syslog.identifier", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_PID:        FieldConversion{"syslog.pid", true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_CGROUP:    FieldConversion{"systemd.cgroup", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_OWNER_UID: FieldConversion{"systemd.owner_uid", true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SESSION:   FieldConversion{"systemd.session", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SLICE:     FieldConversion{"systemd.slice", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT:      FieldConversion{"systemd.unit", false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_USER_UNIT: FieldConversion{"systemd.user_unit", false, false},
		sdjournal.SD_JOURNAL_FIELD_TRANSPORT:         FieldConversion{"systemd.transport", false, false},
		sdjournal.SD_JOURNAL_FIELD_UID:               FieldConversion{"process.uid", true, false},

		// docker journald fields from: https://docs.docker.com/config/containers/logging/journald/
		"CONTAINER_ID":              FieldConversion{"conatiner.id_truncated", false, false},
		"CONTAINER_ID_FULL":         FieldConversion{"container.id", false, false},
		"CONTAINER_NAME":            FieldConversion{"container.name", false, false},
		"CONTAINER_TAG":             FieldConversion{"container.image.tag", false, false},
		"CONTAINER_PARTIAL_MESSAGE": FieldConversion{"container.partial", false, false},

		// dropped fields
		sdjournal.SD_JOURNAL_FIELD_MONOTONIC_TIMESTAMP:       FieldConversion{"", false, true}, // saved in the registry
		sdjournal.SD_JOURNAL_FIELD_SOURCE_REALTIME_TIMESTAMP: FieldConversion{"", false, true}, // saved in the registry
		sdjournal.SD_JOURNAL_FIELD_CURSOR:                    FieldConversion{"", false, true}, // saved in the registry
		"_SOURCE_MONOTONIC_TIMESTAMP":                        FieldConversion{"", false, true}, // received timestamp stored in @timestamp
	}
)
