// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"os"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func TestNewLnkFromPath(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test_lnk_36.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_36.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_48.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_48.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_1332.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_1332.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_1828.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_1828.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_1946.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_1946.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_2404.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_2404.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_2636.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_2636.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_3634.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_3634.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_4008.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_4008.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_4325.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_4325.bin",
			},
			wantErr: false,
		},
		{
			name: "test_lnk_5312.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_5312.bin",
			},
			wantErr: false,
		},
		{
			name: "non_existent_file",
			args: args{
				filePath: "./testdata/lnks/non_existent.bin",
			},
			wantErr: true,
		},
	}

	log := logger.New(os.Stdout, true)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLnkFromPath(tt.args.filePath, log)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLnkFromPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("NewLnkFromPath() returned nil, expected Lnk struct")
			}
		})
	}
}

func TestNewLnkFromBytes(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid_lnk_file",
			args: args{
				data: func() []byte {
					data, _ := os.ReadFile("./testdata/lnks/lnk_36.bin")
					return data
				}(),
			},
			wantErr: false,
		},
		{
			name: "too_short_data",
			args: args{
				data: []byte{0x4c, 0x00},
			},
			wantErr: true,
		},
		{
			name: "invalid_signature",
			args: args{
				data: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
			wantErr: true,
		},
		{
			name: "valid_signature_but_too_short",
			args: args{
				data: []byte{0x4c, 0x00, 0x00, 0x00},
			},
			wantErr: true,
		},
		{
			name: "empty_data",
			args: args{
				data: []byte{},
			},
			wantErr: true,
		},
	}

	log := logger.New(os.Stdout, true)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLnkFromBytes(tt.args.data, log)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLnkFromBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("NewLnkFromBytes() returned nil, expected Lnk struct")
			}
		})
	}
}

