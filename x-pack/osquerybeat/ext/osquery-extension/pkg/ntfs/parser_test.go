// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPartitions(t *testing.T) {
	volumeInfo, err := newVolume("C")
	assert.NoError(t, err, "GetVolumeInformation Failed")

	partitions, err := getPartitions(volumeInfo.Device)
	assert.NoError(t, err, "getPartitions failed")
	assert.NotEmpty(t, partitions, "getPartitions returned no partitions")
	for _, partition := range partitions {
		marshaled, err := json.MarshalIndent(partition, "", "  ")
		assert.NoError(t, err, "Failed to marshal partition to JSON")
		t.Logf("Partition: %s", string(marshaled))
		t.Logf("Raw attributes: %s", partition.AttributesMask)
	}
}

func TestGetAllDriveLetters(t *testing.T) {
	got, gotErr := getAllDriveLetters()
	assert.NoError(t, gotErr, "GetAllDriveLetters() should not return an error")
	for index, value := range got {
		t.Logf("Drive %d: %s", index, value)
	}
	assert.Contains(t, got, "C", "Expected drive C to be present in the list of drive letters")
	assert.NotEmpty(t, got, "GetAllDriveLetters() returned no drive letters")
}
func TestGetVolumeInformation(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		driveLetter string
		want        *Volume
		wantErr     bool
	}{
		{
			name:        "C drive information",
			driveLetter: "C",
			want: &Volume{
				DriveLetter:    "C",
				FileSystemName: "NTFS",
			},
			wantErr: false,
		},
		{
			name:        "C: drive information with colon",
			driveLetter: "C:",
			want: &Volume{
				DriveLetter:    "C",
				FileSystemName: "NTFS",
			},
			wantErr: false,
		},
		{
			name:        "Invalid drive letter",
			driveLetter: "Z",
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := newVolume(tt.driveLetter)
			if gotErr != nil {
				if !tt.wantErr {
					assert.FailNow(t, "GetVolumeInformation Failed")
				}
				return
			}
			if tt.wantErr {
				t.Fatal("GetVolumeInformation() succeeded unexpectedly")
			}
			assert.Equal(t, tt.want.DriveLetter, got.DriveLetter, "Drive letters do not match")
			assert.NotNil(t, got.Device, "Device should not be nil")
			assert.Equal(t, tt.want.FileSystemName, got.FileSystemName, "File system names do not match")
		})
	}
}

func TestVolumeInfo_FindByInode(t *testing.T) {
	tests := []struct {
		name        string
		driveLetter string
		inode       int64
		wantInode   int64
		wantType    string
		wantErr     bool
	}{
		{
			name:        "root directory",
			driveLetter: "C",
			inode:       5,
			wantInode:   5,
			wantType:    "directory",
		},
		{
			name:        "$MFT system file",
			driveLetter: "C",
			inode:       0,
			wantInode:   0,
			wantType:    "file",
		},
		{
			name:        "out-of-range inode",
			driveLetter: "C",
			inode:       math.MaxInt64,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := newVolume(tt.driveLetter)
			require.NoError(t, err, "newVolume(%q) failed", tt.driveLetter)

			got, gotErr := v.FindByInode(tt.inode)
			if tt.wantErr {
				assert.Error(t, gotErr, "FindByInode(%d) should have returned an error", tt.inode)
				return
			}
			require.NoError(t, gotErr, "FindByInode(%d) failed", tt.inode)

			result, err := got.Materialize()
			require.NoError(t, err, "Materialize() failed for inode %d", tt.inode)

			assert.Equal(t, tt.wantInode, result.Inode, "inode mismatch")
			assert.Equal(t, tt.wantType, result.Type, "type mismatch")
			assert.True(t, result.Active, "entry should be active")
			assert.NotEmpty(t, result.Path, "path should not be empty")
		})
	}
}

func TestVolumeInfo_FindByPath(t *testing.T) {
	tests := []struct {
		name         string
		driveLetter  string
		path         string
		wantFilename string
		wantType     string
		wantErr      bool
	}{
		{
			name:         "existing file",
			driveLetter:  "C",
			path:         `C:\Windows\System32\notepad.exe`,
			wantFilename: "notepad.exe",
			wantType:     "file",
		},
		{
			name:         "existing directory",
			driveLetter:  "C",
			path:         `C:\Windows\System32`,
			wantFilename: "System32",
			wantType:     "directory",
		},
		{
			name:        "non-existent path",
			driveLetter: "C",
			path:        `C:\does_not_exist\ghost.txt`,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := newVolume(tt.driveLetter)
			require.NoError(t, err, "newVolume(%q) failed", tt.driveLetter)

			got, gotErr := v.FindByPath(tt.path, nil)
			if tt.wantErr {
				assert.Error(t, gotErr, "FindByPath(%q) should have returned an error", tt.path)
				return
			}
			require.NoError(t, gotErr, "FindByPath(%q) failed", tt.path)

			// Verify the raw MFT entry is valid before materializing.
			assert.NotNil(t, got.mftEntry, "mftEntry should not be nil")
			assert.Positive(t, got.mftEntry.Record_number(), "MFT record number should be non-zero")

			result, err := got.Materialize()
			require.NoError(t, err, "Materialize() failed for path %q", tt.path)

			assert.Equal(t, tt.wantFilename, result.Filename, "filename mismatch")
			assert.Equal(t, tt.wantType, result.Type, "type mismatch")
			assert.True(t, result.Active, "entry should be active")
			assert.Equal(t, tt.path, result.Path, "path mismatch")
			assert.Positive(t, result.Inode, "inode should be non-zero")
		})
	}
}

func TestVolume_explodePath(t *testing.T) {
	tests := []struct {
		name    string
		volume  *Volume
		p       string
		want    []string
		wantErr bool
	}{
		{
			name: "normal path",
			volume: &Volume{
				DriveLetter: "C",
			},
			p: "C:\\Windows\\System32\\notepad.exe",
			want: []string{
				"Windows",
				"System32",
				"notepad.exe",
			},
			wantErr: false,
		},
		{
			name: "Wrong drive letter in path",
			volume: &Volume{
				DriveLetter: "C",
			},
			p:       "D:\\Windows\\System32\\notepad.exe",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty path",
			volume:  &Volume{DriveLetter: "C"},
			p:       "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "root path only (no components after drive letter)",
			volume:  &Volume{DriveLetter: "C"},
			p:       `C:\`,
			want:    nil,
			wantErr: true,
		},
		{
			name:   "path with ADS suffix strips stream name",
			volume: &Volume{DriveLetter: "C"},
			p:      `C:\file.txt:Zone.Identifier`,
			want:   []string{"file.txt"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := tt.volume.explodePath(tt.p)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("explodePath() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("explodePath() succeeded unexpectedly")
			}
			if !assert.Equal(t, tt.want, got, "explodePath() = %v, want %v", got, tt.want) {
				t.Errorf("explodePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
