// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package parsers

import (
	"fmt"
	"os"
	"path/filepath"
    "testing"
	
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/testdata"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)



func TestParseLnk_CustomDestinations(t *testing.T) {
	log := logger.New(os.Stdout, true)

     type args struct {
		filePath string
	 }
	 type want struct {
		description string
		numberOfLnkFiles int
		appId string
		shellItemCounts []int
	 }
	 tests := []struct {
		name string
		args args
		want want
		wantErr bool
	 }{
		{
			name: "TestParseLnk", 
			args: args{
				filePath: "590aee7bdd69b59b.customDestinations-ms",
			},
			want: want{
				description: "Windows Powershell 5.0 64-bit", 
				numberOfLnkFiles: 3,
				appId: "590aee7bdd69b59b",
				shellItemCounts: []int{9, 7, 7},
			},
			wantErr: false,
		},
		{
			name: "TestParseLnk", 
			args: args{
				filePath: "ccba5a5986c77e43.customDestinations-ms",
			},
			want: want{
				description: "Microsoft Edge (Chromium)", 
				numberOfLnkFiles: 2,
				appId: "ccba5a5986c77e43",
				shellItemCounts: []int{7, 7},
			},
			wantErr: false,
		},
		{
			name: "TestParseLnk", 
			args: args{
				filePath: "f4ed0c515fdbcbc.customDestinations-ms",
			},
			want: want{
				description: "", 
				numberOfLnkFiles: 3,
				appId: "f4ed0c515fdbcbc",
				shellItemCounts: []int{6,6,6},
			},
			wantErr: false,
		},
		{
			name: "TestParseLnk", 
			args: args{
				filePath: "ff99ba2fb2e34b73.customDestinations-ms",
			},
			want: want{
				description: "Windows Calculator",
				numberOfLnkFiles: 5,
				appId: "ff99ba2fb2e34b73",
				shellItemCounts: []int{1,1,1,1,1},
			},
			wantErr: false,
		},
	 }
	
	for _, test := range tests {
		for _, filePath := range testdata.MustGetCustomDestinations(t) {
			lnkFiles, gotErr := ParseCustomDestination(filePath, log)
			fmt.Printf("gotErr: %v\n", gotErr)
			fmt.Printf("filePath: %s\n", filePath)
			fmt.Printf("numberOfLnkFiles: %d\n", len(lnkFiles))
			fmt.Printf("lnkFileCount: %d\n", len(lnkFiles))
			for _, lnkFile := range lnkFiles {
				fmt.Printf("  AppId:          %s\n", lnkFile.AppId.Id)
				fmt.Printf("  AppId Name:     %s\n", lnkFile.AppId.Name)
				fmt.Printf("  ShellItemCount: %d\n", len(lnkFile.ShellItems))
			}
		}
		t.Run(test.name, func(t *testing.T) {
			filePath := filepath.Join(testdata.MustGetCustomDestinationDirectory(t), test.args.filePath)
			lnkFiles, gotErr := ParseCustomDestination(filePath, log)
			if gotErr != nil {
				t.Errorf("ParseLnk() failed: %v", gotErr)
			}
			if len(lnkFiles) != test.want.numberOfLnkFiles {
				t.Errorf("expected %d LnkFiles, got %d", test.want.numberOfLnkFiles, len(lnkFiles))
			}
			for _, lnkFile := range lnkFiles {
				if lnkFile.AppId.Id != test.want.appId {
					t.Errorf("expected AppId %s, got %s", test.want.appId, lnkFile.AppId.Id)
				}
			}
			for i, lnkFile := range lnkFiles {
				if len(lnkFile.ShellItems) != test.want.shellItemCounts[i] {
					t.Errorf("expected %d ShellItems, got %d", test.want.shellItemCounts[i], len(lnkFile.ShellItems))
				}
			}
		})
	}
}
