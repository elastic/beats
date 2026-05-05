// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elasticntfspartitions "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/ntfs/elastic_ntfs_partitions"
)

const (
	IOCTL_DISK_GET_DRIVE_LAYOUT_EX = 0x00070050

	// maxPhysicalDrives bounds the \\.\PhysicalDriveN probe range. Windows
	// supports up to 256 disks but real endpoints have far fewer; 128 is
	// ample headroom without making enumeration expensive on sparse setups.
	maxPhysicalDrives = 128
)

// Fixed 48-byte header preceding the variable-length partition array.
// PartitionStyle/PartitionCount are 4 bytes each; the GPT union member
// is the larger at 40 bytes (16-byte GUID + two int64s + uint32 + 4 pad).
type DRIVE_LAYOUT_INFORMATION_EX_HEADER struct {
	PartitionStyle uint32
	PartitionCount uint32
	// GPT union member (largest at 40 bytes):
	DiskId               [16]byte // GUID
	StartingUsableOffset int64
	UsableLength         int64
	MaxPartitionCount    uint32
	_                    [4]byte // padding to 48 bytes; aligns struct to int64 boundary
}

const MBRPartitionStyle = 0
const GPTPartitionStyle = 1
const RAWPartitionStyle = 2

var partitionTypeNames = map[string]string{
	"C12A7328-F81F-11D2-BA4B-00A0C93EC93B": "System",
	"E3C9E316-0B5C-4DB8-817D-F92DF00215AE": "Reserved",
	"EBD0A0A2-B9E5-4433-87C0-68B6B72699C7": "Basic",
	"5808C8AA-7E8F-42E0-85D2-E1E90434CFB3": "LDM Metadata",
	"AF9B60A0-1431-4F62-BC68-3311714A69AD": "LDM Data",
	"DE94BBA4-06D1-4D40-A16A-BFD50179D6AC": "Recovery",
}

// mbrPartitionTypeNames maps the 1-byte MBR partition type to a friendly
// name. Unknown values fall back to their hex representation.
var mbrPartitionTypeNames = map[byte]string{
	0x00: "Empty",
	0x01: "FAT12",
	0x04: "FAT16 <32M",
	0x05: "Extended",
	0x06: "FAT16",
	0x07: "NTFS/exFAT",
	0x0B: "FAT32",
	0x0C: "FAT32 LBA",
	0x0E: "FAT16 LBA",
	0x0F: "Extended LBA",
	0x27: "Windows Recovery",
	0x42: "Dynamic Disk",
	0x82: "Linux Swap",
	0x83: "Linux",
	0x8E: "Linux LVM",
	0xEE: "GPT Protective",
	0xEF: "EFI System",
	0xFD: "Linux RAID",
}

// This needs to be an ordered list in order to make the output
// of gptAttributeNamesFromBitmask deterministic for testing and display purposes
// because go map iteration order is random.
var gptAttributes = []struct {
	bit  uint64
	name string
}{
	{0x0000000000000001, "RequiredPartition"},
	{0x1000000000000000, "ReadOnly"},
	{0x2000000000000000, "ShadowCopy"},
	{0x4000000000000000, "Hidden"},
	{0x8000000000000000, "NoDriveLetter"},
}

func gptAttributeNamesFromBitmask(attributes uint64) []string {
	var names []string
	for _, attr := range gptAttributes {
		if attributes&attr.bit != 0 {
			names = append(names, attr.name)
		}
	}
	return names
}

type PARTITION_INFORMATION_GPT struct {
	PartitionType [16]byte // GUID
	PartitionId   [16]byte // GUID
	Attributes    uint64
	Name          [36]uint16
}

// PARTITION_INFORMATION_MBR mirrors the MBR member of the Windows
// PARTITION_INFORMATION_EX union. It is strictly smaller than
// PARTITION_INFORMATION_GPT, so it safely overlays the union bytes via Mbr().
type PARTITION_INFORMATION_MBR struct {
	PartitionType       byte
	BootIndicator       byte
	RecognizedPartition byte
	_                   byte // alignment padding before HiddenSectors
	HiddenSectors       uint32
	PartitionId         [16]byte // GUID (Win10+; zeroed on older Windows)
}

type PARTITION_INFORMATION_EX struct {
	PartitionStyle   uint32
	_                [4]byte // alignment padding before int64
	StartingOffset   int64
	PartitionLength  int64
	PartitionNumber  uint32
	RewritePartition bool
	_                [3]byte                   // padding
	Gpt              PARTITION_INFORMATION_GPT // union; MBR is smaller and accessed via Mbr()
}

// Mbr reinterprets the union member as MBR partition info. Valid only when
// PartitionStyle is MBR (0).
func (p *PARTITION_INFORMATION_EX) Mbr() *PARTITION_INFORMATION_MBR {
	return (*PARTITION_INFORMATION_MBR)(unsafe.Pointer(&p.Gpt))
}

// MbrSignature reads the MBR disk signature from the header's union member.
// Valid only when PartitionStyle is MBR (0); DiskId overlays the MBR
// DRIVE_LAYOUT_INFORMATION_MBR whose first 4 bytes are the signature.
func (h *DRIVE_LAYOUT_INFORMATION_EX_HEADER) MbrSignature() uint32 {
	return binary.LittleEndian.Uint32(h.DiskId[0:4])
}

type Partition struct {
	Id             string
	Number         uint32
	Style          string
	Type           string
	StartingOffset int64
	Length         int64

	// sqlite doesn't support uint64 and this value can exceed the max int64, so will cast to a hex string for display purposes
	AttributesMask string
	Attributes     string
	Name           string
}

func guidStringFromBytes(b [16]byte) string {
	d1 := binary.LittleEndian.Uint32(b[0:4])
	d2 := binary.LittleEndian.Uint16(b[4:6])
	d3 := binary.LittleEndian.Uint16(b[6:8])
	return fmt.Sprintf("%08X-%04X-%04X-%04X-%012X", d1, d2, d3, b[8:10], b[10:16])
}

func newPartition(partitionInfo *PARTITION_INFORMATION_EX, mbrDiskSignature uint32) (*Partition, error) {
	if partitionInfo == nil {
		return nil, fmt.Errorf("partitionInfo is nil")
	}
	
	p := &Partition{
		Number:         partitionInfo.PartitionNumber,
		StartingOffset: partitionInfo.StartingOffset,
		Length:         partitionInfo.PartitionLength,
	}

	switch partitionInfo.PartitionStyle {
	case GPTPartitionStyle:
		p.Style = "GPT"
		partitionType := guidStringFromBytes(partitionInfo.Gpt.PartitionType)
		if name, ok := partitionTypeNames[partitionType]; ok {
			partitionType = name
		}

		partitionId := guidStringFromBytes(partitionInfo.Gpt.PartitionId)
		attributes := "None"
		attributeNames := gptAttributeNamesFromBitmask(partitionInfo.Gpt.Attributes)
		if len(attributeNames) > 0 {
			attributes = strings.Join(attributeNames, ",")
		}

		p.Type = partitionType
		p.Id = partitionId
		p.AttributesMask = fmt.Sprintf("0x%016X", partitionInfo.Gpt.Attributes)
		p.Attributes = attributes
		p.Name = windows.UTF16ToString(partitionInfo.Gpt.Name[:])
	case MBRPartitionStyle:
		p.Style = "MBR"
		mbr := partitionInfo.Mbr()
		typeName := fmt.Sprintf("0x%02X", mbr.PartitionType)
		if name, ok := mbrPartitionTypeNames[mbr.PartitionType]; ok {
			typeName = name
		}
		p.Type = typeName
		p.Id = fmt.Sprintf("MBR-%08X-%d", mbrDiskSignature, partitionInfo.PartitionNumber)
		p.AttributesMask = fmt.Sprintf("0x%02X", mbr.BootIndicator)
		if mbr.BootIndicator != 0 {
			p.Attributes = "Bootable"
		} else {
			p.Attributes = "None"
		}
	case RAWPartitionStyle:
		p.Style = "RAW"
	default:
		p.Style = "Unknown"
	}

	return p, nil
}

func getPartitions(physicalDrive string) ([]*Partition, error) {
	if partitions, found := getCachedPartitions(physicalDrive); found {
		return partitions, nil
	}

	log := getLogger()
	ptr, err := windows.UTF16PtrFromString(physicalDrive)
	if err != nil {
		return nil, fmt.Errorf("failed to convert device path to UTF16: %w", err)
	}

	handle, err := windows.CreateFile(
		ptr,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return nil, err
	}

	// Defer with a function to bypass errcheck on the CloseHandle since it is ignored intentionally
	defer func() { _ = windows.CloseHandle(handle) }()

	// Allocate enough for the header plus up to 128 partitions, which should
	// be plenty.  Any more and we will log an error.If we run into an issue where customers have more than 128 partitions
	// we can revisit this function
	buf := make([]byte, int(unsafe.Sizeof(DRIVE_LAYOUT_INFORMATION_EX_HEADER{}))+
		128*int(unsafe.Sizeof(PARTITION_INFORMATION_EX{})))

	// Get the drive layout information
	var bytesReturned uint32
	err = windows.DeviceIoControl(
		handle,
		IOCTL_DISK_GET_DRIVE_LAYOUT_EX,
		nil, 0,
		&buf[0], uint32(len(buf)), //nolint:gosec // G115: buf is sized to 128 partitions, well within uint32
		&bytesReturned,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Sanity check that we have enough bytes for the header + partitions
	headerSize := int(unsafe.Sizeof(DRIVE_LAYOUT_INFORMATION_EX_HEADER{}))
	if int(bytesReturned) < headerSize {
		return nil, fmt.Errorf("DeviceIoControl returned %d bytes, need %d for header", bytesReturned, headerSize)
	}
	header := (*DRIVE_LAYOUT_INFORMATION_EX_HEADER)(unsafe.Pointer(&buf[0]))

	const maxPartitions = 128
	partitionCount := int(header.PartitionCount)
	if partitionCount > maxPartitions {
		log.Errorf("GetPartitions(%s): PartitionCount=%d exceeds buffer capacity (%d); truncating",
			physicalDrive, partitionCount, maxPartitions)
		partitionCount = maxPartitions
	}

	partitionSize := int(unsafe.Sizeof(PARTITION_INFORMATION_EX{}))
	requiredBytes := headerSize + partitionCount*partitionSize
	if int(bytesReturned) < requiredBytes {
		return nil, fmt.Errorf("DeviceIoControl returned %d bytes, need %d for header + %d * %d partitions", bytesReturned, headerSize, partitionCount, partitionSize)
	}

	// The MBR disk signature lives in the header's union region and is only
	// meaningful when the drive layout is MBR; it's used to build stable MBR
	// partition ids.
	var mbrDiskSignature uint32
	if header.PartitionStyle == MBRPartitionStyle {
		mbrDiskSignature = header.MbrSignature()
	}

	// parse each partition entry and convert to our Partition struct
	var partitions []*Partition
	for i := range partitionCount {
		offset := headerSize + i*partitionSize
		partitionInfo := (*PARTITION_INFORMATION_EX)(unsafe.Pointer(&buf[offset]))
		partition, err := newPartition(partitionInfo, mbrDiskSignature)
		if err != nil {
			log.Errorf("Failed to parse partition %d: %v", i, err)
			continue
		}
		partitions = append(partitions, partition)
	}
	cachePartitions(physicalDrive, partitions)
	return partitions, nil
}

func uint32ToInt32(value uint32) int32 {
	if value > uint32(math.MaxInt32) {
		getLogger().Errorf("Value %d exceeds max int32, capping to %d", value, math.MaxInt32)
		return math.MaxInt32
	}
	return int32(value)
}

// getPhysicalDrives enumerates \\.\PhysicalDriveN devices exposed by the
// Windows storage stack, independent of drive-letter mounts. Probing by
// path catches raw, offline, hidden, and otherwise unmounted disks that
// getVolumes() cannot see (it only enumerates letters from
// GetLogicalDrives).
func getPhysicalDrives() []string {
	log := getLogger()
	var drives []string
	for i := range maxPhysicalDrives {
		path := fmt.Sprintf(`\\.\PhysicalDrive%d`, i)
		ptr, err := windows.UTF16PtrFromString(path)
		if err != nil {
			continue
		}
		handle, err := windows.CreateFile(
			ptr,
			0,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
			nil,
			windows.OPEN_EXISTING,
			0,
			0,
		)
		if err != nil {
			if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
				log.Infof("GetPhysicalDrives: probe %s: %v", path, err)
			}
			continue
		}
		_ = windows.CloseHandle(handle)
		drives = append(drives, path)
	}
	return drives
}

func partitionsGenerateFunc(_ context.Context, queryContext table.QueryContext, log *logger.Logger, _ *client.ResilientClient) ([]elasticntfspartitions.Result, error) {
	setLogger(log)

	// Build a device -> partition-number -> drive-letter map from mounted
	// volumes. This is purely decoration: getVolumes() only sees disks with
	// mounted letters, so it must not drive which disks are enumerated.
	driveLetterByPartition := make(map[string]map[uint32]string)
	volumes, err := getVolumes()
	if err != nil {
		log.Errorf("Failed to enumerate volumes for drive-letter mapping: %v", err)
	}
	for _, v := range volumes {
		if v.DeviceType != "DISK" {
			continue
		}
		if _, ok := driveLetterByPartition[v.Device]; !ok {
			driveLetterByPartition[v.Device] = make(map[uint32]string)
		}
		driveLetterByPartition[v.Device][v.PartitionNumber] = v.DriveLetter
	}

	var results []elasticntfspartitions.Result
	for _, d := range getPhysicalDrives() {
		partitions, err := getPartitions(d)
		if err != nil {
			log.Errorf("Failed to get partitions for drive %s: %v", d, err)
			continue
		}
		for _, p := range partitions {
			results = append(results, elasticntfspartitions.Result{
				Device:         d,
				DriveLetter:    driveLetterByPartition[d][p.Number],
				Id:             p.Id,
				Number:         int64(p.Number),
				Style:          p.Style,
				Type:           p.Type,
				StartingOffset: p.StartingOffset,
				Length:         p.Length,
				AttributesMask: p.AttributesMask,
				Attributes:     p.Attributes,
				Name:           p.Name,
			})
		}
	}
	return results, nil
}

func init() {
	elasticntfspartitions.RegisterGenerateFunc(partitionsGenerateFunc)
}
