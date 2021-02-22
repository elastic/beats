package lnk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/elastic/beats/v7/libbeat/formats/common"
)

const (
	// location flags
	volumeIDAndLocalBasePath uint32 = 1 << iota
	commonNetworkRelativeLinkAndPathSuffix
)

const (
	// network share flags
	validDevice uint32 = 1 << iota
	validNetType
)

var (
	driveTypes = []string{
		"DRIVE_UNKNOWN",
		"DRIVE_NO_ROOT_DIR",
		"DRIVE_REMOVABLE",
		"DRIVE_FIXED",
		"DRIVE_REMOTE",
		"DRIVE_CDROM",
		"DRIVE_RAMDISK",
	}
	locationFlags = map[uint32]string{
		volumeIDAndLocalBasePath:               "VolumeIDAndLocalBasePath",
		commonNetworkRelativeLinkAndPathSuffix: "CommonNetworkRelativeLinkAndPathSuffix",
	}
	networkShareFlags = map[uint32]string{
		validDevice:  "ValidDevice",
		validNetType: "ValidNetType",
	}
	// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc#432-network-provider-types
	providerTypes = map[uint32]string{
		0x001a0000: "WNNC_NET_AVID",
		0x001b0000: "WNNC_NET_DOCUSPACE",
		0x001c0000: "WNNC_NET_MANGOSOFT",
		0x001d0000: "WNNC_NET_SERNET",
		0x001e0000: "WNNC_NET_RIVERFRONT1",
		0x001f0000: "WNNC_NET_RIVERFRONT2",
		0x00200000: "WNNC_NET_DECORB",
		0x00210000: "WNNC_NET_PROTSTOR",
		0x00220000: "WNNC_NET_FJ_REDIR",
		0x00230000: "WNNC_NET_DISTINCT",
		0x00240000: "WNNC_NET_TWINS",
		0x00250000: "WNNC_NET_RDR2SAMPLE",
		0x00260000: "WNNC_NET_CSC",
		0x00270000: "WNNC_NET_3IN1",
		0x00290000: "WNNC_NET_EXTENDNET",
		0x002a0000: "WNNC_NET_STAC",
		0x002b0000: "WNNC_NET_FOXBAT",
		0x002c0000: "WNNC_NET_YAHOO",
		0x002d0000: "WNNC_NET_EXIFS",
		0x002e0000: "WNNC_NET_DAV",
		0x002f0000: "WNNC_NET_KNOWARE",
		0x00300000: "WNNC_NET_OBJECT_DIRE",
		0x00310000: "WNNC_NET_MASFAX",
		0x00320000: "WNNC_NET_HOB_NFS",
		0x00330000: "WNNC_NET_SHIVA",
		0x00340000: "WNNC_NET_IBMAL",
		0x00350000: "WNNC_NET_LOCK",
		0x00360000: "WNNC_NET_TERMSRV",
		0x00370000: "WNNC_NET_SRT",
		0x00380000: "WNNC_NET_QUINCY",
		0x00390000: "WNNC_NET_OPENAFS",
		0x003a0000: "WNNC_NET_AVID1",
		0x003b0000: "WNNC_NET_DFS",
		0x003c0000: "WNNC_NET_KWNP",
		0x003d0000: "WNNC_NET_ZENWORKS",
		0x003e0000: "WNNC_NET_DRIVEONWEB",
		0x003f0000: "WNNC_NET_VMWARE",
		0x00400000: "WNNC_NET_RSFX",
		0x00410000: "WNNC_NET_MFILES",
		0x00420000: "WNNC_NET_MS_NFS",
		0x00430000: "WNNC_NET_GOOGLE",
	}
)

func parseLocationInfo(header *Header, offset int64, r io.ReaderAt) (*Location, int64, error) {
	// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-shllink/6813269d-0cc8-4be2-933f-e96e8e3412dc
	if !hasFlag(header.rawLinkFlags, hasLinkInfo) {
		return nil, offset, nil
	}
	size, data, err := readU32Data(offset, r)
	if err != nil {
		return nil, 0, err
	}
	if size < 28 {
		return nil, 0, errors.New("invalid location info")
	}
	flags := binary.LittleEndian.Uint32(data[8:12])
	volumeOffset := binary.LittleEndian.Uint32(data[12:16])
	localBasePathOffset := binary.LittleEndian.Uint32(data[16:20])
	networkOffset := binary.LittleEndian.Uint32(data[20:24])
	commonOffset := binary.LittleEndian.Uint32(data[24:28])

	var volume *Volume
	var localBasePath string
	if hasFlag(flags, volumeIDAndLocalBasePath) {
		localBasePath = common.ReadString(data, int(localBasePathOffset))
		if volumeOffset >= size {
			return nil, 0, errors.New("invalid volume offset")
		}
		volume, err = parseVolumeInfo(data[volumeOffset:])
		if err != nil {
			return nil, 0, err
		}
	}

	var networkShare *NetworkShare
	if hasFlag(flags, commonNetworkRelativeLinkAndPathSuffix) {
		if networkOffset >= size {
			return nil, 0, errors.New("invalid network share offset")
		}
		networkShare, err = parseNetworkShareInfo(data[networkOffset:])
		if err != nil {
			return nil, 0, err
		}
	}

	commonPathSuffix := common.ReadString(data, int(commonOffset))

	return &Location{
		Flags:            parseFlags(locationFlags, flags),
		LocalBasePath:    localBasePath,
		CommonPathSuffix: commonPathSuffix,
		Volume:           volume,
		NetworkShare:     networkShare,
	}, offset + int64(size), nil
}

func parseVolumeInfo(data []byte) (*Volume, error) {
	if len(data) < 16 {
		return nil, errors.New("invalid volume info")
	}
	size := binary.LittleEndian.Uint32(data[0:4])
	if uint32(len(data)) < size {
		return nil, errors.New("invalid volume info")
	}
	driveType := binary.LittleEndian.Uint32(data[4:8])
	driveSerialNumber := binary.LittleEndian.Uint32(data[8:12])
	volumeLabelOffset := binary.LittleEndian.Uint32(data[12:16])
	hasUnicodeLabel := volumeLabelOffset == 0x00000014
	var volumeLabel string
	if hasUnicodeLabel {
		if len(data) < 20 {
			return nil, errors.New("invalid volume info")
		}
		volumeLabelOffset = binary.LittleEndian.Uint32(data[16:20])
		volumeLabel = common.ReadUnicode(data, int(volumeLabelOffset))
	} else {
		volumeLabel = common.ReadString(data, int(volumeLabelOffset))
	}

	normalizedDriveType := "DRIVE_UNKNOWN"
	if uint32(len(driveTypes)) > driveType {
		normalizedDriveType = driveTypes[driveType]
	}
	return &Volume{
		DriveType:         normalizedDriveType,
		DriveSerialNumber: fmt.Sprintf("0x%08x", driveSerialNumber),
		VolumeLabel:       volumeLabel,
	}, nil
}

func parseNetworkShareInfo(data []byte) (*NetworkShare, error) {
	if len(data) < 20 {
		return nil, errors.New("invalid network share info")
	}
	size := binary.LittleEndian.Uint32(data[0:4])
	if uint32(len(data)) < size {
		return nil, errors.New("invalid network share info")
	}
	flags := binary.LittleEndian.Uint32(data[4:8])
	shareNameOffset := binary.LittleEndian.Uint32(data[8:12])
	deviceNameOffset := binary.LittleEndian.Uint32(data[12:16])
	providerType := binary.LittleEndian.Uint32(data[16:20])
	normalizedFlags := parseFlags(networkShareFlags, flags)
	var normalizedProviderType string
	if hasFlag(flags, validNetType) {
		if found, ok := providerTypes[providerType]; ok {
			normalizedProviderType = found
		}
	}
	shareName := common.ReadString(data, int(shareNameOffset))
	var deviceName string
	if hasFlag(flags, validDevice) {
		deviceName = common.ReadString(data, int(deviceNameOffset))
	}
	return &NetworkShare{
		Name:         shareName,
		DeviceName:   deviceName,
		Flags:        normalizedFlags,
		ProviderType: normalizedProviderType,
	}, nil
}
