// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	//"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"www.velocidex.com/golang/go-ntfs/parser"
)

func TestGetPartitions(t *testing.T) {
	volumeInfo, err := newVolume("C")
	assert.NoError(t, err, "GetVolumeInformation Failed")

	partitions, err := GetPartitions(volumeInfo.Device)
	assert.NoError(t, err, "GetPartitions Failed")
	assert.NotEmpty(t, partitions, "GetPartitions returned no partitions")
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
				Device:         "\\\\.\\PhysicalDrive0",
				FileSystemName: "NTFS",
			},
			wantErr: false,
		},
		{
			name:        "C: drive information with colon",
			driveLetter: "C:",
			want: &Volume{
				DriveLetter:    "C",
				Device:         "\\\\.\\PhysicalDrive0",
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
			assert.Equal(t, tt.want.Device, got.Device, "Devices do not match")
			assert.Equal(t, tt.want.FileSystemName, got.FileSystemName, "File system names do not match")
		})
	}
}

func TestVolumeInfo_FindByInode(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		driveLetter string
		// Named input parameters for target function.
		inode   int64
		want    *fileNode
		wantErr bool
	}{
		{
			name:        "Find existing file by inode",
			driveLetter: "C",
			inode:       46459, // Replace with a valid inode for testing
			want:        &fileNode{
				// TODO: Fill in the expected fileNode fields.
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := newVolume(tt.driveLetter)
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			got, gotErr := v.FindByInode(tt.inode)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("FindByInode() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("FindByInode() succeeded unexpectedly")
			}
			t.Logf("Got file node: %v\n", got)
		})
	}
}

func TestVolumeInfo_FindByPath(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		driveLetter string
		// Named input parameters for target function.
		path    string
		want    *parser.MFT_ENTRY
		wantErr bool
	}{
		{
			name:        "Find existing file by path",
			driveLetter: "C",
			path:        "C:\\Windows\\System32\\notepad.exe",
			want:        &parser.MFT_ENTRY{
				// TODO: Fill in the expected MFT_ENTRY fields.
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := newVolume(tt.driveLetter)
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			got, gotErr := v.FindByPath(tt.path, nil)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("FindByPath() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("FindByPath() succeeded unexpectedly")
			}
			t.Logf("Got MFT entry: %s\n", got.mftEntry.DebugString())
		})
	}
}

// func TestVolumeInfo_ScopedSearch(t *testing.T) {
// 	tests := []struct {
// 		name string // description of this test case
// 		// Named input parameters for receiver constructor.
// 		driveLetter string
// 		// Named input parameters for target function.
// 		prefix  string
// 		pattern string
// 		want    []string
// 		wantErr bool
// 	}{
// 		{
// 			name: "Scoped search for .exe files in System32",
// 			driveLetter: "C",
// 			prefix: "Windows\\System32",
// 			pattern: "*.exe",
// 			want: []string{
// 				"C:\\Windows\\System32\\notepad.exe",
// 				"C:\\Windows\\System32\\cmd.exe",
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			v, err := newVolume(tt.driveLetter)
// 			if err != nil {
// 				t.Fatalf("could not construct receiver type: %v", err)
// 			}
// 			got, gotErr := v.ScopedSearch(context.Background(), tt.prefix, tt.pattern)
// 			if gotErr != nil {
// 				if !tt.wantErr {
// 					t.Errorf("ScopedSearch() failed: %v", gotErr)
// 				}
// 				return
// 			}
// 			if tt.wantErr {
// 				t.Fatal("ScopedSearch() succeeded unexpectedly")
// 			}

// 			paths := make([]string, len(got))
// 			for i, fileInfo := range got {
// 				paths[i] = fileInfo.BuildFullPath()
// 			}
// 			for _, path := range paths {
// 				t.Logf("Found file: %s", path)
// 			}
// 			for _, expected := range tt.want {
// 				assert.Contains(t, paths, expected, "ScopedSearch() results do not match expected")
// 			}
// 		})
// 	}
// }

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
