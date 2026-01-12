// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func TestCustomJumplists(t *testing.T) {
	type testCase struct {
		name         string
		filePath     string
		expectError  bool
		expectedRows int
	}

	tests := []testCase{
		{
			name:        "test_custom_jumplist_1",
			filePath:    "./testdata/custom/7e4dca80246863e3.customDestinations-ms",
			expectError: true,
		},
		{
			name:         "test_custom_jumplist_2",
			filePath:     "./testdata/custom/590aee7bdd69b59b.customDestinations-ms",
			expectError:  false,
			expectedRows: 3,
		},
		{
			name:         "test_custom_jumplist_3",
			filePath:     "./testdata/custom/ccba5a5986c77e43.customDestinations-ms",
			expectError:  false,
			expectedRows: 2,
		},
		{
			name:         "test_custom_jumplist_4",
			filePath:     "./testdata/custom/f4ed0c515fdbcbc.customDestinations-ms",
			expectError:  false,
			expectedRows: 3,
		},
		{
			name:         "test_custom_jumplist_5",
			filePath:     "./testdata/custom/ff99ba2fb2e34b73.customDestinations-ms",
			expectError:  false,
			expectedRows: 5,
		},
	}

	log := logger.New(os.Stdout, true)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jumplist, err := parseCustomJumplistFile(test.filePath, &UserProfile{Username: "test", Domain: "test", Sid: "test"}, log)
			if test.expectError {
				assert.Error(t, err, "expected error when parsing custom jumplist")
				assert.Nil(t, jumplist, "expected nil jumplist when parsing custom jumplist")
				return
			}
			assert.NoError(t, err, "expected no error when parsing custom jumplist")
			assert.NotNil(t, jumplist, "expected non-nil jumplist when parsing custom jumplist")
			rows := jumplist.ToRows()
			assert.Equal(t, test.expectedRows, len(rows), "expected %d rows in the jumplist", test.expectedRows)
		})
	}
}

func TestLnk(t *testing.T) {
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
				filePath: "./testdata/lnks/lnk_36.bin",
			},
		},
		{
			name: "test_lnk_48.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_48.bin",
			},
		},
		{
			name: "test_lnk_1332.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_1332.bin",
			},
		},
		{
			name: "test_lnk_5312.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_5312.bin",
			},
		},
	}

	log := logger.New(os.Stdout, true)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := os.ReadFile(tt.args.filePath)
			assert.NoError(t, err, "expected no error when reading LNK file")
			got, err := newLnkFromBytes(bytes, log)
			if tt.wantErr {
				assert.Error(t, err, "expected error when parsing LNK file")
				assert.Nil(t, got, "expected nil LNK when parsing LNK file")
				return
			}
			assert.NoError(t, err, "expected no error when parsing LNK file")
			assert.NotNil(t, got, "expected non-nil LNK when parsing LNK file")
		})
	}
}

func TestGetUserProfiles(t *testing.T) {
	log := logger.New(os.Stdout, true)
	userProfiles, err := getUserProfiles(log)
	assert.NoError(t, err, "expected no error when getting user profiles")
	assert.NotEmpty(t, userProfiles, "expected non-empty user profiles")
}

func TestGetJumplists(t *testing.T) {
	log := logger.New(os.Stdout, true)
	userProfiles, err := getUserProfiles(log)
	assert.NoError(t, err, "expected no error when getting user profiles")
	for _, userProfile := range userProfiles {
		jumplists := userProfile.getJumplists(log)
		for _, jumplist := range jumplists {
			log.Infof("found jumplist: %s, username: %s, domain: %s, sid: %s", jumplist.Path, userProfile.Username, userProfile.Domain, userProfile.Sid)
		}
	}
}
