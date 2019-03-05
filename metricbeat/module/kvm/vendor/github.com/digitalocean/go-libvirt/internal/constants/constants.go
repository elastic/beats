// Copyright 2016 The go-libvirt Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package constants provides shared data for the libvirt package. This file
// includes only things not generated automatically by the parser that runs on
// libvirt's remote_protocol.x file - see constants.gen.go for the generated
// definitions.
package constants

// qemu constants
const (
	ProgramQEMU      = 0x20008087
	ProgramKeepAlive = 0x6b656570
)

// qemu procedure identifiers
const (
	QEMUDomainMonitor                       = 1
	QEMUConnectDomainMonitorEventRegister   = 4
	QEMUConnectDomainMonitorEventDeregister = 5
	QEMUDomainMonitorEvent                  = 6
)

const (
	// PacketLengthSize is the packet length, in bytes.
	PacketLengthSize = 4

	// HeaderSize is the packet header size, in bytes.
	HeaderSize = 24

	// UUIDSize is the length of a UUID, in bytes.
	UUIDSize = 16

	// TypedParamFieldLength is VIR_TYPED_PARAM_FIELD_LENGTH, and is the maximum
	// length of the Field string in virTypedParameter structs.
	TypedParamFieldLength = 80
)
