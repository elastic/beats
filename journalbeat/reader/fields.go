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

import "github.com/coreos/go-systemd/v22/sdjournal"

type fieldConversion struct {
	names     []string
	isInteger bool
	dropped   bool
}

var (
	journaldEventFields = map[string]fieldConversion{
		// provided by systemd journal
		"COREDUMP_UNIT":                              fieldConversion{[]string{"journald.coredump.unit"}, false, false},
		"COREDUMP_USER_UNIT":                         fieldConversion{[]string{"journald.coredump.user_unit"}, false, false},
		"OBJECT_AUDIT_LOGINUID":                      fieldConversion{[]string{"journald.object.audit.login_uid"}, true, false},
		"OBJECT_AUDIT_SESSION":                       fieldConversion{[]string{"journald.object.audit.session"}, true, false},
		"OBJECT_CMDLINE":                             fieldConversion{[]string{"journald.object.cmd"}, false, false},
		"OBJECT_COMM":                                fieldConversion{[]string{"journald.object.name"}, false, false},
		"OBJECT_EXE":                                 fieldConversion{[]string{"journald.object.executable"}, false, false},
		"OBJECT_GID":                                 fieldConversion{[]string{"journald.object.gid"}, true, false},
		"OBJECT_PID":                                 fieldConversion{[]string{"journald.object.pid"}, true, false},
		"OBJECT_SYSTEMD_OWNER_UID":                   fieldConversion{[]string{"journald.object.systemd.owner_uid"}, true, false},
		"OBJECT_SYSTEMD_SESSION":                     fieldConversion{[]string{"journald.object.systemd.session"}, false, false},
		"OBJECT_SYSTEMD_UNIT":                        fieldConversion{[]string{"journald.object.systemd.unit"}, false, false},
		"OBJECT_SYSTEMD_USER_UNIT":                   fieldConversion{[]string{"journald.object.systemd.user_unit"}, false, false},
		"OBJECT_UID":                                 fieldConversion{[]string{"journald.object.uid"}, true, false},
		"_KERNEL_DEVICE":                             fieldConversion{[]string{"journald.kernel.device"}, false, false},
		"_KERNEL_SUBSYSTEM":                          fieldConversion{[]string{"journald.kernel.subsystem"}, false, false},
		"_SYSTEMD_INVOCATION_ID":                     fieldConversion{[]string{"systemd.invocation_id"}, false, false},
		"_SYSTEMD_USER_SLICE":                        fieldConversion{[]string{"systemd.user_slice"}, false, false},
		"_UDEV_DEVLINK":                              fieldConversion{[]string{"journald.kernel.device_symlinks"}, false, false}, // TODO aggregate multiple elements
		"_UDEV_DEVNODE":                              fieldConversion{[]string{"journald.kernel.device_node_path"}, false, false},
		"_UDEV_SYSNAME":                              fieldConversion{[]string{"journald.kernel.device_name"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_AUDIT_LOGINUID:    fieldConversion{[]string{"process.audit.login_uid"}, true, false},
		sdjournal.SD_JOURNAL_FIELD_AUDIT_SESSION:     fieldConversion{[]string{"process.audit.session"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_BOOT_ID:           fieldConversion{[]string{"host.boot_id"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_CAP_EFFECTIVE:     fieldConversion{[]string{"process.capabilites"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_CMDLINE:           fieldConversion{[]string{"process.cmd"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_CODE_FILE:         fieldConversion{[]string{"journald.code.file"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_CODE_FUNC:         fieldConversion{[]string{"journald.code.func"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_CODE_LINE:         fieldConversion{[]string{"journald.code.line"}, true, false},
		sdjournal.SD_JOURNAL_FIELD_COMM:              fieldConversion{[]string{"process.name"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_EXE:               fieldConversion{[]string{"process.executable"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_GID:               fieldConversion{[]string{"process.uid"}, true, false},
		sdjournal.SD_JOURNAL_FIELD_HOSTNAME:          fieldConversion{[]string{"host.hostname"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_MACHINE_ID:        fieldConversion{[]string{"host.id"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_MESSAGE:           fieldConversion{[]string{"message"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_PID:               fieldConversion{[]string{"process.pid"}, true, false},
		sdjournal.SD_JOURNAL_FIELD_PRIORITY:          fieldConversion{[]string{"syslog.priority", "log.syslog.priority"}, true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_FACILITY:   fieldConversion{[]string{"syslog.facility", "log.syslog.facility.name"}, true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER: fieldConversion{[]string{"syslog.identifier"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_PID:        fieldConversion{[]string{"syslog.pid"}, true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_CGROUP:    fieldConversion{[]string{"systemd.cgroup"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_OWNER_UID: fieldConversion{[]string{"systemd.owner_uid"}, true, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SESSION:   fieldConversion{[]string{"systemd.session"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SLICE:     fieldConversion{[]string{"systemd.slice"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT:      fieldConversion{[]string{"systemd.unit"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_USER_UNIT: fieldConversion{[]string{"systemd.user_unit"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_TRANSPORT:         fieldConversion{[]string{"systemd.transport"}, false, false},
		sdjournal.SD_JOURNAL_FIELD_UID:               fieldConversion{[]string{"process.uid"}, true, false},

		// docker journald fields from: https://docs.docker.com/config/containers/logging/journald/
		"CONTAINER_ID":              fieldConversion{[]string{"container.id_truncated"}, false, false},
		"CONTAINER_ID_FULL":         fieldConversion{[]string{"container.id"}, false, false},
		"CONTAINER_NAME":            fieldConversion{[]string{"container.name"}, false, false},
		"CONTAINER_TAG":             fieldConversion{[]string{"container.log.tag"}, false, false},
		"CONTAINER_PARTIAL_MESSAGE": fieldConversion{[]string{"container.partial"}, false, false},

		// dropped fields
		sdjournal.SD_JOURNAL_FIELD_MONOTONIC_TIMESTAMP:       fieldConversion{nil, false, true}, // saved in the registry
		sdjournal.SD_JOURNAL_FIELD_SOURCE_REALTIME_TIMESTAMP: fieldConversion{nil, false, true}, // saved in the registry
		sdjournal.SD_JOURNAL_FIELD_CURSOR:                    fieldConversion{nil, false, true}, // saved in the registry
		"_SOURCE_MONOTONIC_TIMESTAMP":                        fieldConversion{nil, false, true}, // received timestamp stored in @timestamp
	}
)
