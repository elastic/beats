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
		// provided by systemd journal
		"COREDUMP_UNIT":                              "journald.coredump.unit",
		"COREDUMP_USER_UNIT":                         "journald.coredump.user_unit",
		"OBJECT_AUDIT_LOGINUID":                      "journald.object.audit.login_uid",
		"OBJECT_AUDIT_SESSION":                       "journald.object.audit.session",
		"OBJECT_CMDLINE":                             "journald.object.cmd",
		"OBJECT_COMM":                                "journald.object.name",
		"OBJECT_EXE":                                 "journald.object.executable",
		"OBJECT_GID":                                 "journald.object.gid",
		"OBJECT_PID":                                 "journald.object.pid",
		"OBJECT_SYSTEMD_OWNER_UID":                   "journald.object.systemd.owner_uid",
		"OBJECT_SYSTEMD_SESSION":                     "journald.object.systemd.session",
		"OBJECT_SYSTEMD_UNIT":                        "journald.object.systemd.unit",
		"OBJECT_SYSTEMD_USER_UNIT":                   "journald.object.systemd.user_unit",
		"OBJECT_UID":                                 "journald.object.uid",
		"_KERNEL_DEVICE":                             "journald.kernel.device",
		"_KERNEL_SUBSYSTEM":                          "journald.kernel.subsystem",
		"_SYSTEMD_INVOCATION_ID":                     "systemd.invocation_id",
		"_SYSTEMD_USER_SLICE":                        "systemd.user_slice",
		"_UDEV_DEVLINK":                              "journald.kernel.device_symlinks", // TODO aggregate multiple elements
		"_UDEV_DEVNODE":                              "journald.kernel.device_node_path",
		"_UDEV_SYSNAME":                              "journald.kernel.device_name",
		sdjournal.SD_JOURNAL_FIELD_AUDIT_LOGINUID:    "process.audit.login_uid",
		sdjournal.SD_JOURNAL_FIELD_AUDIT_SESSION:     "process.audit.session",
		sdjournal.SD_JOURNAL_FIELD_BOOT_ID:           "host.boot_id",
		sdjournal.SD_JOURNAL_FIELD_CAP_EFFECTIVE:     "process.capabilites",
		sdjournal.SD_JOURNAL_FIELD_CMDLINE:           "process.cmd",
		sdjournal.SD_JOURNAL_FIELD_CODE_FILE:         "journald.code.file",
		sdjournal.SD_JOURNAL_FIELD_CODE_FUNC:         "journald.code.func",
		sdjournal.SD_JOURNAL_FIELD_CODE_LINE:         "journald.code.line",
		sdjournal.SD_JOURNAL_FIELD_COMM:              "process.name",
		sdjournal.SD_JOURNAL_FIELD_EXE:               "process.executable",
		sdjournal.SD_JOURNAL_FIELD_GID:               "process.uid",
		sdjournal.SD_JOURNAL_FIELD_HOSTNAME:          "host.name",
		sdjournal.SD_JOURNAL_FIELD_MACHINE_ID:        "host.id",
		sdjournal.SD_JOURNAL_FIELD_MESSAGE:           "message",
		sdjournal.SD_JOURNAL_FIELD_PID:               "process.pid",
		sdjournal.SD_JOURNAL_FIELD_PRIORITY:          "syslog.priority",
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_FACILITY:   "syslog.facility",
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER: "syslog.identifier",
		sdjournal.SD_JOURNAL_FIELD_SYSLOG_PID:        "syslog.pid",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_CGROUP:    "systemd.cgroup",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_OWNER_UID: "systemd.owner_uid",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SESSION:   "systemd.session",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_SLICE:     "systemd.slice",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT:      "systemd.unit",
		sdjournal.SD_JOURNAL_FIELD_SYSTEMD_USER_UNIT: "systemd.user_unit",
		sdjournal.SD_JOURNAL_FIELD_TRANSPORT:         "systemd.transport",
		sdjournal.SD_JOURNAL_FIELD_UID:               "process.uid",

		// docker journald fields from: https://docs.docker.com/config/containers/logging/journald/
		"CONTAINER_ID":              "conatiner.id_truncated",
		"CONTAINER_ID_FULL":         "container.id",
		"CONTAINER_NAME":            "container.name",
		"CONTAINER_TAG":             "container.image.tag",
		"CONTAINER_PARTIAL_MESSAGE": "container.partial",

		// dropped fields
		sdjournal.SD_JOURNAL_FIELD_MONOTONIC_TIMESTAMP:       "", // saved in the registry
		sdjournal.SD_JOURNAL_FIELD_SOURCE_REALTIME_TIMESTAMP: "", // saved in the registry
		sdjournal.SD_JOURNAL_FIELD_CURSOR:                    "", // saved in the registry
		"_SOURCE_MONOTONIC_TIMESTAMP":                        "", // received timestamp stored in @timestamp
	}
)
