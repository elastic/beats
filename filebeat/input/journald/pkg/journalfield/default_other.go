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

//go:build !linux || !cgo
// +build !linux !cgo

package journalfield

// journaldEventFields provides default field mappings and conversions rules.
var journaldEventFields = FieldConversion{
	// provided by systemd journal
	"COREDUMP_UNIT":            text("journald.coredump.unit"),
	"COREDUMP_USER_UNIT":       text("journald.coredump.user_unit"),
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
	"_KERNEL_DEVICE":           text("journald.kernel.device"),
	"_KERNEL_SUBSYSTEM":        text("journald.kernel.subsystem"),
	"_SYSTEMD_INVOCATION_ID":   text("systemd.invocation_id"),
	"_SYSTEMD_USER_SLICE":      text("systemd.user_slice"),
	"_UDEV_DEVLINK":            text("journald.kernel.device_symlinks"),
	"_UDEV_DEVNODE":            text("journald.kernel.device_node_path"),
	"_UDEV_SYSNAME":            text("journald.kernel.device_name"),
}
