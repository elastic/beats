// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// windowsVolumeReader wraps a raw volume handle as an io.ReaderAt.
type windowsVolumeReader struct {
	handle windows.Handle
}

// ReadAt reads from the volume at the specified offset. The offset must be sector-aligned (512-byte boundaries).
func (r *windowsVolumeReader) ReadAt(p []byte, off int64) (int, error) {
	// Windows raw volume reads must be sector-aligned (512-byte boundaries).
	// go-ntfs's PagedReader handles alignment for us, but the underlying
	// ReadAt still needs to issue aligned reads to the kernel.
	var done uint32
	overlapped := &windows.Overlapped{
		Offset:     uint32(off & 0xFFFFFFFF),
		OffsetHigh: uint32(off >> 32), //nolint:gosec // G115: shifted to high 32 bits
	}
	err := windows.ReadFile(r.handle, p, &done, overlapped)
	if err != nil {
		return int(done), err
	}
	return int(done), nil
}

// Close closes the volume handle.
func (r *windowsVolumeReader) Close() {
	err := windows.CloseHandle(r.handle)
	if err != nil {
		getLogger().Errorf("failed to close handle: %v\n", err)
	}
}

// NewVolumeReader opens a handle to the specified drive letter (e.g. "C") and returns a windowsVolumeReader for it.
// The drive letter is normalized and validated  (e.g. "C", "c", "C:", "c:", "C:\", "c:\" are all accepted and normalized to "C").
func NewVolumeReader(driveLetter string) (*windowsVolumeReader, error) {
	driveLetter, err := normalizeDriveLetter(driveLetter)
	if err != nil {
		return nil, err
	}
	path := `\\.\` + driveLetter + `:`
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_NO_BUFFERING, // required for raw sector reads
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateFile(%s): %w", path, err)
	}
	return &windowsVolumeReader{handle: handle}, nil
}
