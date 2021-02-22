package lnk

// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc

import (
	"io"
	"time"
)

// Console contains LNK extra console data block info
type Console struct {
	FillAttributes         []string `json:"fillAttributes,omitempty"`
	PopupFillAttributes    []string `json:"popupFillAttributes,omitempty"`
	ScreenBufferSizeX      uint16   `json:"screenBufferSizeX"`
	ScreenBufferSizeY      uint16   `json:"screenBufferSizeY"`
	WindowSizeX            uint16   `json:"windowSizeX"`
	WindowSizeY            uint16   `json:"windowSizeY"`
	WindowOriginX          uint16   `json:"windowOriginX"`
	WindowOriginY          uint16   `json:"windowOriginY"`
	FontSize               uint32   `json:"fontSize"`
	FontFamily             string   `json:"fontFamily,omitempty"`
	FontWeight             uint32   `json:"fontWeight"`
	FaceName               string   `json:"faceName,omitempty"`
	CursorSize             uint32   `json:"cursorSize"`
	FullScreen             bool     `json:"fullScreen"`
	QuickEdit              bool     `json:"quickEdit"`
	InsertMode             bool     `json:"insertMode"`
	AutoPosition           bool     `json:"autoPosition"`
	HistoryBufferSize      uint32   `json:"historyBufferSize"`
	NumberOfHistoryBuffers uint32   `json:"numberOfHistoryBuffers"`
	HistoryNoDup           bool     `json:"historyNoDup"`
	ColorTable             []string `json:"colorTable"`
}

// ConsoleFE contains LNK extra console data block info
type ConsoleFE struct {
	CodePage string `json:"codePage"`
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
	NamedProperties map[string][]Property `json:"namedProperties,omitempty"`
	Properties      map[uint32][]Property `json:"properties,omitempty"`
}

// Shim contains LNK extra shim data block info
type Shim struct {
	LayerName string `json:"layerName,omitempty"`
}

// SpecialFolder contains LNK extra special folder data block info
type SpecialFolder struct {
	ID     uint32 `json:"id"`
	Offset uint32 `json:"offset"`
}

// Tracker contains LNK extra tracker data block info
type Tracker struct {
	Version    uint32   `json:"version"`
	MachineID  string   `json:"machineId"`
	Droid      []string `json:"droid,omitempty"`
	DroidBirth []string `json:"droidBirth,omitempty"`
}

// VistaAndAboveIDList contains LNK extra vista and above id list data block info
type VistaAndAboveIDList struct {
	Targets []Target `json:"targets,omitempty"`
}

// Extra contains LNK extra block info
type Extra struct {
	// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-shllink/c41e062d-f764-4f13-bd4f-ea812ab9a4d1
	Console             *Console             `json:"console,omitempty"`
	ConsoleFE           *ConsoleFE           `json:"consoleFE,omitempty"`
	Darwin              *Darwin              `json:"darwin,omitempty"`
	Environment         *Environment         `json:"environment,omitempty"`
	IconEnvironment     *IconEnvironment     `json:"iconEnvironment,omitempty"`
	KnownFolder         *KnownFolder         `json:"knownFolder,omitempty"`
	PropertyStore       *PropertyStore       `json:"propertyStore,omitempty"`
	Shim                *Shim                `json:"shim,omitempty"`
	SpecialFolder       *SpecialFolder       `json:"specialFolder,omitempty"`
	Tracker             *Tracker             `json:"tracker,omitempty"`
	VistaAndAboveIDList *VistaAndAboveIDList `json:"vistaAndAboveIdList,omitempty"`
}

// Volume contains LNK location volume info
type Volume struct {
	// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc#42-volume-information
	DriveType         string `json:"driveType,omitempty"`
	DriveSerialNumber string `json:"driveSerialNumber,omitempty"`
	VolumeLabel       string `json:"volumeLabel,omitempty"`
}

// NetworkShare contains LNK location network share info
type NetworkShare struct {
	// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc#43-network-share-information
	Flags        []string `json:"flags,omitempty"`
	ProviderType string   `json:"providerType,omitempty"`
	Name         string   `json:"name,omitempty"`
	DeviceName   string   `json:"deviceName,omitempty"`
}

// Location contains LNK location info
type Location struct {
	// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-shllink/6813269d-0cc8-4be2-933f-e96e8e3412dc
	// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc#4-location-information
	Flags            []string `json:"flags"`
	CommonPathSuffix string   `json:"commonPathSuffix,omitempty"`
	// Location information data
	Volume        *Volume `json:"volume,omitempty"`
	LocalBasePath string  `json:"localBasePath,omitempty"`
	// The network share information
	NetworkShare *NetworkShare `json:"networkShare,omitempty"`
}

// Target contains LNK target info
type Target struct {
	Size   uint16 `json:"size"`
	TypeID uint8  `json:"typeId"`
	SHA256 string `json:"sha256"`
}

// Header contains LNK header info
type Header struct {
	GUID         string     `json:"guid"`
	LinkFlags    []string   `json:"linkFlags"`
	FileFlags    []string   `json:"fileFlags"`
	CreationTime *time.Time `json:"creationTime,omitempty"`
	AccessedTime *time.Time `json:"accessedTime,omitempty"`
	ModfiedTime  *time.Time `json:"modifiedTime,omitempty"`
	FileSize     uint32     `json:"fileSize,omitempty"`
	IconIndex    uint32     `json:"iconIndex"`
	WindowStyle  string     `json:"windowStyle"`
	HotKey       string     `json:"hotKey,omitempty"`

	rawLinkFlags uint32
	rawFileFlags uint32
}

// Info contains high level fingerprinting an analysis of an LNK file.
type Info struct {
	Header           *Header   `json:"header"`
	Targets          []Target  `json:"targets,omitempty"`
	Location         *Location `json:"location,omitempty"`
	Name             string    `json:"name,omitempty"`
	RelativePath     string    `json:"relativePath,omitempty"`
	WorkingDirectory string    `json:"workingDirectory,omitempty"`
	CommandLine      string    `json:"commandLine,omitempty"`
	IconLocation     string    `json:"iconLocation,omitempty"`
	Extra            *Extra    `json:"extra,omitempty"`
}

// Parse parses the LNK file and returns information about it or errors.
func Parse(r io.ReaderAt) (*Info, error) {
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
