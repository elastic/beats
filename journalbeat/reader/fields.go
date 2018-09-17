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

var (
	journaldEventFields = map[string]string{
		"COREDUMP_UNIT":                              "coredump.unit",
		"COREDUMP_USER_UNIT":                         "coredump.user_unit",
		"OBJECT_AUDIT_LOGINUID":                      "object.audit.login_uid",
		"OBJECT_AUDIT_SESSION":                       "object.audit.session",
		"OBJECT_CMDLINE":                             "object.cmd",
		"OBJECT_COMM":                                "object.name",
		"OBJECT_EXE":                                 "object.executable",
		"OBJECT_GID":                                 "object.gid",
		"OBJECT_PID":                                 "object.pid",
		"OBJECT_SYSTEMD_OWNER_UID":                   "object.systemd.owner_uid",
		"OBJECT_SYSTEMD_SESSION":                     "object.systemd.session",
		"OBJECT_SYSTEMD_UNIT":                        "object.systemd.unit",
		"OBJECT_SYSTEMD_USER_UNIT":                   "object.systemd.user_unit",
		"OBJECT_UID":                                 "object.uid",
		"_KERNEL_DEVICE":                             "kernel.device",
		"_KERNEL_SUBSYSTEM":                          "kernel.subsystem",
		"_SYSTEMD_INVOCATION_ID":                     "sytemd.invocation_id",
		"_UDEV_DEVLINK":                              "kernel.device_symlinks", // TODO aggregate multiple elements
		"_UDEV_DEVNODE":                              "kernel.device_node_path",
		"_UDEV_SYSNAME":                              "kernel.device_name",
		sdjournal.SD_JOURNAL_FIELD_AUDIT_LOGINUID:    "process.audit.login_uid",
		sdjournal.SD_JOURNAL_FIELD_AUDIT_SESSION:     "process.audit.session",
		sdjournal.SD_JOURNAL_FIELD_BOOT_ID:           "host.boot_id",
		sdjournal.SD_JOURNAL_FIELD_CMDLINE:           "process.cmd",
		sdjournal.SD_JOURNAL_FIELD_CODE_FILE:         "code.file",
		sdjournal.SD_JOURNAL_FIELD_CODE_FUNC:         "code.func",
		sdjournal.SD_JOURNAL_FIELD_CODE_LINE:         "code.line",
		sdjournal.SD_JOURNAL_FIELD_COMM:              "process.name",
		sdjournal.SD_JOURNAL_FIELD_EXE:               "process.executable",
		sdjournal.SD_JOURNAL_FIELD_GID:               "process.uid",
		sdjournal.SD_JOURNAL_FIELD_HOSTNAME:          "host.name",
		sdjournal.SD_JOURNAL_FIELD_MACHINE_ID:        "host.id",
		sdjournal.SD_JOURNAL_FIELD_PID:               "process.pid",
		sdjournal.SD_JOURNAL_FIELD_PRIORITY:          "syslog.priority",
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_FACILITY:   "syslog.facility",
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER: "syslog.identifier",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_CGROUP:    "systemd.cgroup",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_OWNER_UID: "systemd.owner_uid",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SESSION:   "systemd.session",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SLICE:     "systemd.slice",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT:      "systemd.unit",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_USER_UNIT: "systemd.user_unit",
		sdjournal.SD_JOURNAL_FIELD_TRANSPORT:         "systemd.transport",
		sdjournal.SD_JOURNAL_FIELD_UID:               "process.uid",
		sdjournal.SD_JOURNAL_FIELD_MESSAGE:           "message",
	}
)
