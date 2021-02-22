package lnk

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

const (
	// link flags
	hasTargetIDList uint32 = 1 << iota
	hasLinkInfo
	hasName
	hasRelativePath
	hasWorkingDir
	hasArguments
	hasIconLocation
	isUnicode
	forceNoLinkInfo
	hasExpString
	runInSeparateProcess
	_
	hasDarwinID
	runAsUser
	hasExpIcon
	noPidlAlias
	_
	runWithShimLayer
	forceNoLinkTrack
	enableTargetMetadata
	disableLinkPathTracking
	disableKnownFolderTracking
	disableKnownFolderAlias
	allowLinkToLink
	unaliasOnSave
	preferEnvironmentPath
	keepLocalIDListForUNCTarget
)

const (
	// file flags
	fileAttributeReadonly uint32 = 1 << iota
	fileAttributeHidden
	fileAttributeSystem
	_
	fileAttributeDirectory
	fileAttributeArchive
	fileAttributeDevice
	fileAttributeNormal
	fileAttributeTemporary
	fileAttributeSparseFile
	fileAttributeReparsePoint
	fileAttributeCompressed
	fileAttributeOffline
	fileAttributeNotContentIndexed
	fileAttributeEncrypted
	_
	fileAttributeVirtual
)

var (
	windowStyles = []string{
		"SW_HIDE",
		"SW_NORMAL",
		"SW_SHOWMINIMIZED",
		"SW_MAXIMIZE ",
		"SW_SHOWNOACTIVATE",
		"SW_SHOW",
		"SW_MINIMIZE",
		"SW_SHOWMINNOACTIVE",
		"SW_SHOWNA",
		"SW_RESTORE",
		"SW_SHOWDEFAULT",
		"SW_FORCEMINIMIZE",
	}
	hotKeyModifiers = []string{
		"UNSET",
		"HOTKEYF_SHIFT",
		"HOTKEYF_CONTROL",
		"HOTKEYF_ALT",
	}
	fKeys = []string{
		"VK_F1",
		"VK_F2",
		"VK_F3",
		"VK_F4",
		"VK_F5",
		"VK_F6",
		"VK_F7",
		"VK_F8",
		"VK_F9",
		"VK_F10",
		"VK_F11",
		"VK_F12",
		"VK_F13",
		"VK_F14",
		"VK_F15",
		"VK_F16",
		"VK_F17",
		"VK_F18",
		"VK_F19",
		"VK_F20",
		"VK_F21",
		"VK_F22",
		"VK_F23",
		"VK_F24",
	}
	linkFlags = map[uint32]string{
		hasTargetIDList:             "HasTargetIDList",
		hasLinkInfo:                 "HasLinkInfo",
		hasName:                     "HasName",
		hasRelativePath:             "HasRelativePath",
		hasWorkingDir:               "HasWorkingDir",
		hasArguments:                "HasArguments",
		hasIconLocation:             "HasIconLocation",
		isUnicode:                   "IsUnicode",
		forceNoLinkInfo:             "ForceNoLinkInfo",
		hasExpString:                "HasExpString",
		runInSeparateProcess:        "RunInSeparateProcess",
		hasDarwinID:                 "HasDarwinID",
		runAsUser:                   "RunAsUser",
		hasExpIcon:                  "HasExpIcon",
		noPidlAlias:                 "NoPidlAlias",
		runWithShimLayer:            "RunWithShimLayer",
		forceNoLinkTrack:            "ForceNoLinkTrack",
		enableTargetMetadata:        "EnableTargetMetadata",
		disableLinkPathTracking:     "DisableLinkPathTracking",
		disableKnownFolderTracking:  "DisableKnownFolderTracking",
		disableKnownFolderAlias:     "DisableKnownFolderAlias",
		allowLinkToLink:             "AllowLinkToLink",
		unaliasOnSave:               "UnaliasOnSave",
		preferEnvironmentPath:       "PreferEnvironmentPath",
		keepLocalIDListForUNCTarget: "KeepLocalIDListForUNCTarget",
	}
	fileFlags = map[uint32]string{
		fileAttributeReadonly:          "FILE_ATTRIBUTE_READONLY",
		fileAttributeHidden:            "FILE_ATTRIBUTE_HIDDEN",
		fileAttributeSystem:            "FILE_ATTRIBUTE_SYSTEM",
		fileAttributeDirectory:         "FILE_ATTRIBUTE_DIRECTORY",
		fileAttributeArchive:           "FILE_ATTRIBUTE_ARCHIVE",
		fileAttributeDevice:            "FILE_ATTRIBUTE_DEVICE",
		fileAttributeNormal:            "FILE_ATTRIBUTE_NORMAL",
		fileAttributeTemporary:         "FILE_ATTRIBUTE_TEMPORARY",
		fileAttributeSparseFile:        "FILE_ATTRIBUTE_SPARSE_FILE",
		fileAttributeReparsePoint:      "FILE_ATTRIBUTE_REPARSE_POINT",
		fileAttributeCompressed:        "FILE_ATTRIBUTE_COMPRESSED",
		fileAttributeOffline:           "FILE_ATTRIBUTE_OFFLINE",
		fileAttributeNotContentIndexed: "FILE_ATTRIBUTE_NOT_CONTENT_INDEXED",
		fileAttributeEncrypted:         "FILE_ATTRIBUTE_ENCRYPTED",
		fileAttributeVirtual:           "FILE_ATTRIBUTE_VIRTUAL",
	}
)

// 116444736000000000 is the number of 100-nanoseconds between
// 1 january 1601 00:00 and 1 january 1970 00:00 UTC
const epochDelta uint64 = 116444736000000000

func windowsTimeToUnix(timestamp uint64) uint64 {
	// Convert to 100-nanoseconds increment since Unix Epoch and then
	// truncate to seconds
	return (timestamp - epochDelta) / 1e7
}

func parseHeader(r io.ReaderAt) (*Header, int64, error) {
	header := make([]byte, 76)
	read, err := r.ReadAt(header, 0)
	if err != nil {
		return nil, 0, err
	}
	if read != 76 {
		return nil, 0, errors.New("truncated LNK header")
	}
	rawLinkFlags := binary.LittleEndian.Uint32(header[20:24])
	rawFileFlags := binary.LittleEndian.Uint32(header[24:28])
	return &Header{
		GUID:         encodeUUID(header[4:20]),
		rawLinkFlags: rawLinkFlags,
		LinkFlags:    parseFlags(linkFlags, rawLinkFlags),
		rawFileFlags: rawFileFlags,
		FileFlags:    parseFlags(fileFlags, rawFileFlags),
		CreationTime: normalizeTime(binary.LittleEndian.Uint64(header[28:36])),
		AccessedTime: normalizeTime(binary.LittleEndian.Uint64(header[36:44])),
		ModfiedTime:  normalizeTime(binary.LittleEndian.Uint64(header[44:52])),
		FileSize:     binary.LittleEndian.Uint32(header[52:56]),
		IconIndex:    binary.LittleEndian.Uint32(header[56:60]),
		WindowStyle:  normalizeWindowStyle(binary.LittleEndian.Uint32(header[60:64])),
		HotKey:       normalizeHotKey(header[64], header[65]),
	}, 76, nil
}

func normalizeWindowStyle(style uint32) string {
	if style >= uint32(len(windowStyles)) {
		return fmt.Sprintf("UNKNOWN:%d", style)
	}
	return windowStyles[style]
}

func normalizeTime(value uint64) *time.Time {
	if value == 0 {
		return nil
	}
	timestamp := time.Unix(int64(windowsTimeToUnix(value)), 0).UTC()
	return &timestamp
}

func normalizeHotKey(lower, upper uint8) string {
	if lower == 0x00 && upper == 0x00 {
		return ""
	}
	var key string
	if upper < uint8(len(hotKeyModifiers)) {
		modifier := hotKeyModifiers[upper]
		if modifier != "UNSET" {
			key = modifier + "+"
		}
	}
	if (0x30 <= lower && lower <= 0x39) || (0x41 <= lower && lower <= 0x5a) {
		return key + string(rune(lower))
	}
	if (lower - 0x70) < uint8(len(fKeys)) {
		return key + fKeys[lower-0x70]
	}
	if lower == 0x90 {
		return key + "VK_NUMLOCK"
	}
	if lower == 0x91 {
		return key + "VK_SCROLL"
	}
	return "UNKNOWN"
}

func parseFlags(flagset map[uint32]string, value uint32) []string {
	flags := []string{}
	for flag, name := range flagset {
		if hasFlag(value, flag) {
			flags = append(flags, name)
		}
	}
	sort.Strings(flags)
	return flags
}

func encodeUUID(uuid []byte) string {
	dst := make([]byte, 36)
	hex.Encode(dst, uuid[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], uuid[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], uuid[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], uuid[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], uuid[10:])
	return string(dst)
}

func hasFlag(flagset, flag uint32) bool {
	return (flagset & flag) != 0
}
