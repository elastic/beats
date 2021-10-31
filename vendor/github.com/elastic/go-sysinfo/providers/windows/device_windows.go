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

package windows

import (
	"strings"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

const (
	// DeviceMup is the device used for unmounted network filesystems
	DeviceMup = "\\device\\mup"

	// LANManRedirector is an string that appears in mounted network filesystems
	LANManRedirector = "lanmanredirector"
)

var (
	// ErrNoDevice is the error returned by DevicePathToDrivePath when
	// an invalid device-path is supplied.
	ErrNoDevice = errors.New("not a device path")

	// ErrDeviceNotFound is the error returned by DevicePathToDrivePath when
	// a path pointing to an unmapped device is passed.
	ErrDeviceNotFound = errors.New("logical device not found")
)

type deviceProvider interface {
	GetLogicalDrives() (uint32, error)
	QueryDosDevice(*uint16, *uint16, uint32) (uint32, error)
}

type deviceMapper struct {
	deviceProvider
}

type winapiDeviceProvider struct{}

type testingDeviceProvider map[byte]string

func newDeviceMapper() deviceMapper {
	return deviceMapper{
		deviceProvider: winapiDeviceProvider{},
	}
}

func fixNetworkDrivePath(device string) string {
	// For a VirtualBox share:
	// device=\device\vboxminirdr\;z:\vboxsvr\share
	// path=\device\vboxminirdr\vboxsvr\share
	//
	// For a network share:
	// device=\device\lanmanredirector\;q:nnnnnnn\server\share
	// path=\device\mup\server\share

	semicolonPos := strings.IndexByte(device, ';')
	colonPos := strings.IndexByte(device, ':')
	if semicolonPos == -1 || colonPos != semicolonPos+2 {
		return device
	}
	pathStart := strings.IndexByte(device[colonPos+1:], '\\')
	if pathStart == -1 {
		return device
	}
	dev := device[:semicolonPos]
	path := device[colonPos+pathStart+1:]
	n := len(dev)
	if n > 0 && dev[n-1] == '\\' {
		dev = dev[:n-1]
	}
	return dev + path
}

func (mapper *deviceMapper) getDevice(driveLetter byte) (string, error) {
	driveBuf := [3]uint16{uint16(driveLetter), ':', 0}

	for bufSize := 64; bufSize <= 1024; bufSize *= 2 {
		deviceBuf := make([]uint16, bufSize)
		n, err := mapper.QueryDosDevice(&driveBuf[0], &deviceBuf[0], uint32(len(deviceBuf)))
		if err != nil {
			if err == windows.ERROR_INSUFFICIENT_BUFFER {
				continue
			}
			return "", err
		}
		return windows.UTF16ToString(deviceBuf[:n]), nil
	}
	return "", windows.ERROR_INSUFFICIENT_BUFFER
}

func (mapper *deviceMapper) DevicePathToDrivePath(path string) (string, error) {
	pathLower := strings.ToLower(path)
	isMUP := strings.Index(pathLower, DeviceMup) == 0
	mask, err := mapper.GetLogicalDrives()
	if err != nil {
		return "", errors.Wrap(err, "GetLogicalDrives")
	}

	for bit := uint32(0); mask != 0 && bit < uint32('Z'-'A'+1); bit++ {
		if mask&(1<<bit) == 0 {
			continue
		}
		mask ^= 1 << bit
		driveLetter := byte('A' + bit)
		dev, err := mapper.getDevice(driveLetter)
		if err != nil {
			continue
		}

		dev = fixNetworkDrivePath(strings.ToLower(dev))
		found := strings.Index(pathLower, dev) == 0

		if !found && isMUP && strings.Contains(dev, LANManRedirector) {
			dev = strings.Replace(dev, LANManRedirector, "mup", 1)
			found = strings.Index(pathLower, dev) == 0
		}
		if found {
			off := len(dev)
			if off < len(path) && path[off] == '\\' {
				off++
			}
			return string(driveLetter) + ":\\" + path[off:], nil
		}
	}
	// Handle unmapped shares:
	// \device\mup\server\share\path -> \\server\share\path
	if isMUP {
		return "\\" + path[len(DeviceMup):], nil
	}
	return "", ErrDeviceNotFound
}

func (winapiDeviceProvider) GetLogicalDrives() (uint32, error) {
	return windows.GetLogicalDrives()
}

func (winapiDeviceProvider) QueryDosDevice(name *uint16, buf *uint16, length uint32) (uint32, error) {
	return windows.QueryDosDevice(name, buf, length)
}

func (m testingDeviceProvider) GetLogicalDrives() (mask uint32, err error) {
	for drive := range m {
		mask |= 1 << uint32(drive-'A')
	}
	return mask, nil
}

func ptrOffset(ptr *uint16, off uint32) *uint16 {
	return (*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(off*2)))
}

func (m testingDeviceProvider) QueryDosDevice(nameW *uint16, buf *uint16, length uint32) (uint32, error) {
	drive := byte(*nameW)
	if byte(*ptrOffset(nameW, 1)) != ':' {
		return 0, errors.New("not a drive")
	}
	if *ptrOffset(nameW, 2) != 0 {
		return 0, errors.New("drive not terminated")
	}
	path, ok := m[drive]
	if !ok {
		return 0, errors.Errorf("drive %c not found", drive)
	}
	n := uint32(len(path))
	if n+2 > length {
		return 0, windows.ERROR_INSUFFICIENT_BUFFER
	}
	for i := uint32(0); i < n; i++ {
		*ptrOffset(buf, i) = uint16(path[i])
	}
	*ptrOffset(buf, n) = 0
	*ptrOffset(buf, n+1) = 0
	return n + 2, nil
}
