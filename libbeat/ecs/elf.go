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

import (
	"time"
)

// These fields contain Linux Executable Linkable Format (ELF) metadata.
type Elf struct {
	// Extracted when possible from the file's metadata. Indicates when it was
	// built or compiled. It can also be faked by malware creators.
	CreationDate time.Time `ecs:"creation_date"`

	// Machine architecture of the ELF file.
	Architecture string `ecs:"architecture"`

	// Byte sequence of ELF file.
	ByteOrder string `ecs:"byte_order"`

	// CPU type of the ELF file.
	CpuType string `ecs:"cpu_type"`

	// Header class of the ELF file.
	HeaderClass string `ecs:"header.class"`

	// Data table of the ELF header.
	HeaderData string `ecs:"header.data"`

	// Application Binary Interface (ABI) of the Linux OS.
	HeaderOsAbi string `ecs:"header.os_abi"`

	// Header type of the ELF file.
	HeaderType string `ecs:"header.type"`

	// Version of the ELF header.
	HeaderVersion string `ecs:"header.version"`

	// Version of the ELF Application Binary Interface (ABI).
	HeaderAbiVersion string `ecs:"header.abi_version"`

	// Header entrypoint of the ELF file.
	HeaderEntrypoint int64 `ecs:"header.entrypoint"`

	// "0x1" for original ELF files.
	HeaderObjectVersion string `ecs:"header.object_version"`

	// An array containing an object for each section of the ELF file.
	// The keys that should be present in these objects are defined by
	// sub-fields underneath `elf.sections.*`.
	Sections []Sections `ecs:"sections"`

	// List of exported element names and types.
	Exports map[string]interface{} `ecs:"exports"`

	// List of imported element names and types.
	Imports map[string]interface{} `ecs:"imports"`

	// List of shared libraries used by this ELF object.
	SharedLibraries string `ecs:"shared_libraries"`

	// telfhash symbol hash for ELF file.
	Telfhash string `ecs:"telfhash"`

	// An array containing an object for each segment of the ELF file.
	// The keys that should be present in these objects are defined by
	// sub-fields underneath `elf.segments.*`.
	Segments []Segments `ecs:"segments"`
}

type Sections struct {
	// ELF Section List flags.
	Flags string `ecs:"flags"`

	// ELF Section List name.
	Name string `ecs:"name"`

	// ELF Section List offset.
	PhysicalOffset string `ecs:"physical_offset"`

	// ELF Section List type.
	Type string `ecs:"type"`

	// ELF Section List physical size.
	PhysicalSize int64 `ecs:"physical_size"`

	// ELF Section List virtual address.
	VirtualAddress int64 `ecs:"virtual_address"`

	// ELF Section List virtual size.
	VirtualSize int64 `ecs:"virtual_size"`

	// Shannon entropy calculation from the section.
	Entropy int64 `ecs:"entropy"`

	// Chi-square probability distribution of the section.
	Chi2 int64 `ecs:"chi2"`
}

type Segments struct {
	// ELF object segment type.
	Type string `ecs:"type"`

	// ELF object segment sections.
	Sections string `ecs:"sections"`
}
