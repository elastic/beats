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

package lnk

// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc

import (
	"io"
	"time"
)

// Console contains LNK extra console data block info
type Console struct {
	FillAttributes         []string `json:"fill_attributes,omitempty"`
	PopupFillAttributes    []string `json:"popup_fill_attributes,omitempty"`
	ScreenBufferSizeX      uint16   `json:"screen_buffer_size_x"`
	ScreenBufferSizeY      uint16   `json:"screen_buffer_size_y"`
	WindowSizeX            uint16   `json:"window_size_x"`
	WindowSizeY            uint16   `json:"window_size_y"`
	WindowOriginX          uint16   `json:"window_origin_x"`
	WindowOriginY          uint16   `json:"window_origin_y"`
	FontSize               uint32   `json:"font_size"`
	FontFamily             string   `json:"font_family,omitempty"`
	FontWeight             uint32   `json:"font_weight"`
	FaceName               string   `json:"face_name,omitempty"`
	CursorSize             uint32   `json:"cursor_size"`
	FullScreen             bool     `json:"full_screen"`
	QuickEdit              bool     `json:"quick_edit"`
	InsertMode             bool     `json:"insert_mode"`
	AutoPosition           bool     `json:"auto_position"`
	HistoryBufferSize      uint32   `json:"history_buffer_size"`
	NumberOfHistoryBuffers uint32   `json:"number_of_history_buffers"`
	HistoryNoDup           bool     `json:"history_no_dup"`
	ColorTable             []string `json:"color_table"`
}

// ConsoleFE contains LNK extra console data block info
type ConsoleFE struct {
	CodePage string `json:"code_page"`
}

// Darwin contains LNK extra darwin data block info
type Darwin struct {
	ANSI    string `json:"ansi"`
	Unicode string `json:"unicode"`
}

// Environment contains LNK extra environment data block info
type Environment struct {
	ANSI    string `json:"ansi"`
	Unicode string `json:"unicode"`
}

// IconEnvironment contains LNK extra icon environment data block info
type IconEnvironment struct {
	ANSI    string `json:"ansi"`
	Unicode string `json:"unicode"`
}

// KnownFolder contains LNK extra known folder data block info
type KnownFolder struct {
	ID     string `json:"id"`
	Offset uint32 `json:"offset"`
}

// Property contains property storage propery info
type Property struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// PropertyStore contains LNK extra property store data block info
type PropertyStore struct {
	// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-propstore/3453fb82-0e4f-4c2c-bc04-64b4bd2c51ec
	NamedProperties map[string][]Property `json:"named_properties,omitempty"`
	Properties      map[uint32][]Property `json:"properties,omitempty"`
}

// Shim contains LNK extra shim data block info
type Shim struct {
	LayerName string `json:"layer_name,omitempty"`
}

// SpecialFolder contains LNK extra special folder data block info
type SpecialFolder struct {
	ID     uint32 `json:"id"`
	Offset uint32 `json:"offset"`
}

// Tracker contains LNK extra tracker data block info
type Tracker struct {
	Version    uint32   `json:"version"`
	MachineID  string   `json:"machine_id"`
	Droid      []string `json:"droid,omitempty"`
	DroidBirth []string `json:"droid_birth,omitempty"`
}

// VistaAndAboveIDList contains LNK extra vista and above id list data block info
type VistaAndAboveIDList struct {
	Targets []Target `json:"targets,omitempty"`
}

// Extra contains LNK extra block info
type Extra struct {
	// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-shllink/c41e062d-f764-4f13-bd4f-ea812ab9a4d1
	Console             *Console             `json:"console,omitempty"`
	ConsoleFE           *ConsoleFE           `json:"console_fe,omitempty"`
	Darwin              *Darwin              `json:"darwin,omitempty"`
	Environment         *Environment         `json:"environment,omitempty"`
	IconEnvironment     *IconEnvironment     `json:"icon_environment,omitempty"`
	KnownFolder         *KnownFolder         `json:"known_folder,omitempty"`
	PropertyStore       *PropertyStore       `json:"property_store,omitempty"`
	Shim                *Shim                `json:"shim,omitempty"`
	SpecialFolder       *SpecialFolder       `json:"special_folder,omitempty"`
	Tracker             *Tracker             `json:"tracker,omitempty"`
	VistaAndAboveIDList *VistaAndAboveIDList `json:"vista_and_above_id_list,omitempty"`
}

// Volume contains LNK location volume info
type Volume struct {
	// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc#42-volume-information
	DriveType         string `json:"drive_type,omitempty"`
	DriveSerialNumber string `json:"drive_serial_number,omitempty"`
	VolumeLabel       string `json:"volume_label,omitempty"`
}

// NetworkShare contains LNK location network share info
type NetworkShare struct {
	// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc#43-network-share-information
	Flags        []string `json:"flags,omitempty"`
	ProviderType string   `json:"provider_type,omitempty"`
	Name         string   `json:"name,omitempty"`
	DeviceName   string   `json:"device_name,omitempty"`
}

// Location contains LNK location info
type Location struct {
	// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-shllink/6813269d-0cc8-4be2-933f-e96e8e3412dc
	// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc#4-location-information
	Flags            []string `json:"flags"`
	CommonPathSuffix string   `json:"common_path_suffix,omitempty"`
	// Location information data
	Volume        *Volume `json:"volume,omitempty"`
	LocalBasePath string  `json:"local_base_path,omitempty"`
	// The network share information
	NetworkShare *NetworkShare `json:"network_share,omitempty"`
}

// Target contains LNK target info
type Target struct {
	Size   uint16 `json:"size"`
	TypeID uint8  `json:"type_id"`
	SHA256 string `json:"sha256"`
}

// Header contains LNK header info
type Header struct {
	GUID         string     `json:"guid"`
	LinkFlags    []string   `json:"link_flags"`
	FileFlags    []string   `json:"file_flags"`
	CreationTime *time.Time `json:"creation_time,omitempty"`
	AccessedTime *time.Time `json:"accessed_time,omitempty"`
	ModifiedTime *time.Time `json:"modified_time,omitempty"`
	FileSize     uint32     `json:"file_size,omitempty"`
	IconIndex    uint32     `json:"icon_index"`
	WindowStyle  string     `json:"window_style"`
	HotKey       string     `json:"hot_key,omitempty"`

	rawLinkFlags uint32
	rawFileFlags uint32
}

// Info contains high level fingerprinting an analysis of an LNK file.
type Info struct {
	Header           *Header   `json:"header"`
	Targets          []Target  `json:"targets,omitempty"`
	Location         *Location `json:"location,omitempty"`
	Name             string    `json:"name,omitempty"`
	RelativePath     string    `json:"relative_path,omitempty"`
	WorkingDirectory string    `json:"working_directory,omitempty"`
	CommandLine      string    `json:"command_line,omitempty"`
	IconLocation     string    `json:"icon_location,omitempty"`
	Extra            *Extra    `json:"extra,omitempty"`
}

// Parse parses the LNK file and returns information about it or errors.
func Parse(r io.ReaderAt) (interface{}, error) {
	header, offset, err := parseHeader(r)
	if err != nil {
		return nil, err
	}
	targets, offset, err := parseTargets(header, offset, r)
	if err != nil {
		return nil, err
	}
	location, offset, err := parseLocationInfo(header, offset, r)
	if err != nil {
		return nil, err
	}
	name, offset, err := readDataString(header, hasName, offset, r)
	if err != nil {
		return nil, err
	}
	relativePath, offset, err := readDataString(header, hasRelativePath, offset, r)
	if err != nil {
		return nil, err
	}
	workingDirectory, offset, err := readDataString(header, hasWorkingDir, offset, r)
	if err != nil {
		return nil, err
	}
	commandLine, offset, err := readDataString(header, hasArguments, offset, r)
	if err != nil {
		return nil, err
	}
	iconLocation, offset, err := readDataString(header, hasIconLocation, offset, r)
	if err != nil {
		return nil, err
	}
	extra, err := parseExtraBlocks(header, offset, r)
	if err != nil {
		return nil, err
	}
	return &Info{
		Header:           header,
		Targets:          targets,
		Location:         location,
		Name:             name,
		RelativePath:     relativePath,
		WorkingDirectory: workingDirectory,
		CommandLine:      commandLine,
		IconLocation:     iconLocation,
		Extra:            extra,
	}, nil
}
