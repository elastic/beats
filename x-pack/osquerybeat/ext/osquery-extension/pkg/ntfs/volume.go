// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/osquery/osquery-go/plugin/table"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elasticntfsvolumes "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/ntfs/elastic_ntfs_volumes"
)

const (
	// IOCTL_STORAGE_GET_DEVICE_NUMBER is used to get the physical drive number for a given
	// volume, which is needed to correlate volumes with partitions.
	IOCTL_STORAGE_GET_DEVICE_NUMBER = 0x2d1080
)

// deviceTypeMap maps Windows STORAGE_DEVICE_NUMBER.DeviceType values to human-readable strings.
// see: https://learn.microsoft.com/en-us/windows-hardware/drivers/kernel/specifying-device-types
var deviceTypeMap = map[uint32]string{
	0x00000002: "CD_ROM",
	0x00000007: "DISK",
	0x0000002d: "MASS_STORAGE",
	0x00000024: "VIRTUAL_DISK",
	0x00000033: "DVD",
}

// STORAGE_DEVICE_NUMBER structure as defined in Windows API
type STORAGE_DEVICE_NUMBER struct {
	DeviceType      uint32
	DeviceNumber    uint32
	PartitionNumber uint32
}

// Volume represents a mounted volume in the system, such as C:\ or D:\. It contains
// basic information about the volume and a lazily initialized NTFS session for NTFS-specific operations.
type Volume struct {
	Device          string
	DeviceType      string
	DriveLetter     string
	VolumeLabel     string
	FileSystemName  string
	PartitionNumber uint32

	// ntfsSession is lazily initialized when NTFS-specific operations are needed.
	// It will be nil for non-NTFS volumes.
	sessionNeedsClose atomic.Bool
	ntfsSession       func() (*NTFSSession, error)
}

// normalizeDriveLetter validates and normalizes a drive letter input, ensuring it is a single uppercase character.
func normalizeDriveLetter(driveLetter string) (string, error) {
	d := strings.TrimSpace(driveLetter)
	d = strings.TrimSuffix(d, `\`)
	d = strings.TrimSuffix(d, ":")
	if len(d) != 1 {
		return "", fmt.Errorf("invalid drive letter %q", driveLetter)
	}
	return strings.ToUpper(d), nil
}

// Close releases any resources associated with the volume, such as the NTFS session if it was initialized.
func (v *Volume) Close() {
	if v.sessionNeedsClose.Load() {
		session, err := v.ntfsSession()
		if err == nil && session != nil {
			session.Close()
		}
	}
}

// getAllDriveLetters retrieves a list of all drive letters currently in use
// on the system by querying the logical drive bitmask.
func getAllDriveLetters() ([]string, error) {
	bitmask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, err
	}
	driveLetters := make([]string, 0)
	for i := range uint(26) {
		if bitmask&(1<<i) == 0 {
			continue
		}
		driveLetters = append(driveLetters, string(rune('A'+i)))
	}
	return driveLetters, nil
}

// newVolume creates a Volume struct for the given drive letter
// by querying the Windows API for volume information.
func newVolume(driveLetter string) (*Volume, error) {
	// Normalize and validate the drive letter input
	driveLetter, err := normalizeDriveLetter(driveLetter)
	if err != nil {
		return nil, err
	}
	// Construct the device path for the volume, e.g. \\.\C:
	path := `\\.\` + driveLetter + `:`
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("failed to convert device path to UTF16: %w", err)
	}

	// Initialize the Volume struct with the drive letter. Other fields will be populated after querying the API.
	volumeInfo := &Volume{
		DriveLetter: driveLetter,
	}

	// Open handle to the logical drive
	handle, err := windows.CreateFile(
		ptr,
		0, // No access needed just to query device info
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	// Defer with a function to bypass errcheck on the CloseHandle since it is ignored intentionally
	defer func() { _ = windows.CloseHandle(handle) }()

	// Query the device number
	var sdn STORAGE_DEVICE_NUMBER
	var bytesReturned uint32
	err = windows.DeviceIoControl(
		handle,
		IOCTL_STORAGE_GET_DEVICE_NUMBER,
		nil,
		0,
		(*byte)(unsafe.Pointer(&sdn)),
		uint32(unsafe.Sizeof(sdn)),
		&bytesReturned,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to device io control: %w", err)
	}

	if deviceType, ok := deviceTypeMap[sdn.DeviceType]; ok {
		volumeInfo.DeviceType = deviceType
	} else {
		volumeInfo.DeviceType = fmt.Sprintf("Unknown (0x%X)", sdn.DeviceType)
	}

	if volumeInfo.DeviceType == "DISK" {
		volumeInfo.Device = fmt.Sprintf(`\\.\PhysicalDrive%d`, sdn.DeviceNumber)
	} else {
		volumeInfo.Device = fmt.Sprintf(`\\.\%s:`, driveLetter)
	}

	volumeInfo.PartitionNumber = sdn.PartitionNumber

	// Query the volume information
	var volumeName [windows.MAX_PATH + 1]uint16
	var fsName [windows.MAX_PATH + 1]uint16
	volumePath := driveLetter + `:\`
	ptr, err = windows.UTF16PtrFromString(volumePath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert volume path to UTF16: %w", err)
	}

	err = windows.GetVolumeInformation(ptr, &volumeName[0], uint32(len(volumeName)), nil, nil, nil, &fsName[0], uint32(len(fsName)))
	if err != nil {
		return nil, fmt.Errorf("failed to get volume information: %w", err)
	}
	volumeInfo.VolumeLabel = windows.UTF16ToString(volumeName[:])
	volumeInfo.FileSystemName = windows.UTF16ToString(fsName[:])

	// Initialize the NTFS session lazily, since it can be expensive to open a handle to the volume and read the MFT.
	// and we don't need to do so, if for example the user is querying the volumes table which only needs basic volume information.
	volumeInfo.ntfsSession = sync.OnceValues(func() (*NTFSSession, error) {
		if volumeInfo.FileSystemName != "NTFS" {
			return nil, fmt.Errorf("volume %s is not NTFS", volumeInfo.DriveLetter)
		}
		session, err := newNTFSSession(volumeInfo.DriveLetter)
		if err != nil {
			return nil, err
		}
		volumeInfo.sessionNeedsClose.Store(true)
		return session, err
	})

	return volumeInfo, nil
}

// getVolumes retrieves information about all mounted volumes in the system and returns a slice of Volume structs.
func getVolumes() ([]*Volume, error) {
	log := getLogger()
	driveLetters, err := getAllDriveLetters()
	if err != nil {
		return nil, err
	}

	var volumes []*Volume

	for _, driveLetter := range driveLetters {
		// Check the cache first to avoid expensive volume initialization if we've already seen this drive letter recently.
		if volume, found := getCachedVolumes(driveLetter); found {
			volumes = append(volumes, volume)
			continue
		}

		volume, err := newVolume(driveLetter)
		if err != nil {
			log.Infof("failed to get volume information for drive %s: %v", driveLetter, err)
			continue
		}
		cacheVolume(driveLetter, volume)
		volumes = append(volumes, volume)
	}
	return volumes, nil
}

func volumesGenerateFunc(_ context.Context, _ table.QueryContext, log *logger.Logger, _ *client.ResilientClient) ([]elasticntfsvolumes.Result, error) {
	setLogger(log)
	volumes, err := getVolumes()
	if err != nil {
		return nil, err
	}

	results := make([]elasticntfsvolumes.Result, 0, len(volumes))
	for _, v := range volumes {
		results = append(results, elasticntfsvolumes.Result{
			Device:         v.Device,
			DeviceType:     v.DeviceType,
			DriveLetter:    v.DriveLetter,
			VolumeLabel:    v.VolumeLabel,
			FileSystemName: v.FileSystemName,
		})
	}
	return results, nil
}

func init() {
	elasticntfsvolumes.RegisterGenerateFunc(volumesGenerateFunc)
}
