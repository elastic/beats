// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package lnk

import (
	"os"
	"fmt"
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
		want    *Lnk
		wantErr bool
	}{
		{
			name: "test_lnk_36.bin",
			args: args{
				filePath: "../../testdata/lnks/lnk_36.bin",
			},
		},
		{
			name: "test_lnk_48.bin",
			args: args{
				filePath: "../../testdata/lnks/lnk_48.bin",
			},
		},
		{
			name: "test_lnk_1332.bin",
			args: args{
				filePath: "../../testdata/lnks/lnk_1332.bin",
			},
		},
	}
    log := logger.New(os.Stdout, true)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLnkFromPath(tt.args.filePath, log)
			if err != nil {
				t.Errorf("NewLnkFromPath() error = %v", err)
				return
			}
			fmt.Printf("Lnk: %v\n", got)
		})
	}
}
