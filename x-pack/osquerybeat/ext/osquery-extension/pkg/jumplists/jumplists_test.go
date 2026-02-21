// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"os"
	"path/filepath"
	"testing"

	osquerygen "github.com/osquery/osquery-go/gen/osquery"
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
			jumplist, err := parseCustomJumplistFile(test.filePath, &UserProfile{Username: "test", Sid: "test"}, log)
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

func TestGetColumns(t *testing.T) {
	columns := GetColumns()
	assert.NotNil(t, columns, "expected non-nil columns")
	assert.Greater(t, len(columns), 0, "expected at least 1 column")
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

type MockClient struct {
	t *testing.T
}

func (m *MockClient) Query(sql string) (*osquerygen.ExtensionResponse, error) {
	_ = sql
	profileDir := m.t.TempDir()
	recentDir := filepath.Join(profileDir, "AppData", "Roaming", "Microsoft", "Windows", "Recent")
	assert.NoError(m.t, os.MkdirAll(recentDir, 0o755))

	customJumplistDir := filepath.Join(recentDir, "CustomDestinations")
	assert.NoError(m.t, os.MkdirAll(customJumplistDir, 0o755))
	bytes, err := os.ReadFile("./testdata/custom/590aee7bdd69b59b.customDestinations-ms")
	assert.NoError(m.t, err, "expected no error when reading custom jumplist test file")
	assert.NoError(m.t, os.WriteFile(filepath.Join(customJumplistDir, "590aee7bdd69b59b.customDestinations-ms"), bytes, 0o644))

	automaticJumplistDir := filepath.Join(recentDir, "AutomaticDestinations")
	assert.NoError(m.t, os.MkdirAll(automaticJumplistDir, 0o755))
	bytes, err = os.ReadFile("./testdata/automatic/4db07e3587413f4d.automaticDestinations-ms")
	assert.NoError(m.t, err, "expected no error when reading automatic jumplist test file")
	assert.NoError(m.t, os.WriteFile(filepath.Join(automaticJumplistDir, "4db07e3587413f4d.automaticDestinations-ms"), bytes, 0o644))

	return &osquerygen.ExtensionResponse{
		Response: []map[string]string{
			{
				"username":  "testuser",
				"uuid":      "S-1-5-21-1234567890-123456789-1234567890-1001",
				"directory": profileDir,
			},
		},
	}, nil
}

func TestAutomaticJumpList(t *testing.T) {
	type testCase struct {
		name        string
		filePath    string
		expectError bool
	}
	tests := []testCase{
		{
			name:        "test_olecfb_2",
			filePath:    "./testdata/automatic/5f7b5f1e01b83767.automaticDestinations-ms",
			expectError: false,
		},
		{
			name:        "test_olecfb_3",
			filePath:    "./testdata/automatic/6cbc8013911ed22e.automaticDestinations-ms",
			expectError: false,
		},
		{
			name:        "test_olecfb_4",
			filePath:    "./testdata/automatic/7e4dca80246863e3.automaticDestinations-ms",
			expectError: false,
		},
		{
			name:        "test_olecfb_5",
			filePath:    "./testdata/automatic/9b9cdc69c1c24e2b.automaticDestinations-ms",
			expectError: false,
		},
		{
			name:        "test_olecfb_7",
			filePath:    "./testdata/automatic/47c6675663a92f2a.automaticDestinations-ms",
			expectError: false,
		},
		{
			name:        "test_olecfb_8",
			filePath:    "./testdata/automatic/607c8cee3ce959c.automaticDestinations-ms",
			expectError: false,
		},
		{
			name:        "test_olecfb_11",
			filePath:    "./testdata/automatic/befe8a0a7d3eeb43.automaticDestinations-ms",
			expectError: false,
		},
		{
			name:        "test_olecfb_12",
			filePath:    "./testdata/automatic/ccba5a5986c77e43.automaticDestinations-ms",
			expectError: false,
		},
		{
			name:        "test_olecfb_15",
			filePath:    "./testdata/automatic/db9172b310c92fa6.automaticDestinations-ms",
			expectError: false,
		},
		{
			name:        "test_olecfb_16",
			filePath:    "./testdata/automatic/f01b4d95cf55d32a.automaticDestinations-ms",
			expectError: false,
		},
	}
	log := logger.New(os.Stdout, true)
	for _, test := range tests {
		automaticJumpList, err := ParseAutomaticJumpListFile(test.filePath, &UserProfile{Username: "test", Sid: "test"}, log)
		if err != nil {
			t.Fatalf("%s %s ParseAutomaticJumpListFile() returned error: %v", test.filePath, test.name, err)
		}
		if test.expectError {
			assert.Error(t, err, "expected error when parsing Automatic Jump List")
			assert.Nil(t, automaticJumpList, "expected nil Automatic Jump List when parsing Automatic Jump List")
			return
		}
		assert.NoError(t, err, "expected no error when parsing Automatic Jump List")
		assert.NotNil(t, automaticJumpList, "expected non-nil Automatic Jump List when parsing Automatic Jump List")
		rows := automaticJumpList.ToRows()
		assert.Greater(t, len(rows), 0, "expected at least 1 row in the Automatic Jump List")

		// If an automatic jumplist has only one row, it could mean that the jumplist is empty.
		// or it could mean that the jumplist has only one entry.  If the jumplist is empty,
		// both the DestListEntry and the Lnk will be nil.  If the jumplist has only one entry,
		// the DestListEntry will be non-nil and the Lnk will be non-nil.
		if len(rows) == 1 && rows[0].DestListEntry == nil {
			assert.Nil(t, rows[0].Lnk, "expected nil LNK when parsing Automatic Jump List")
		} else {
			for _, row := range rows {
				assert.NotNil(t, row.Lnk, "expected non-nil LNK when parsing Automatic Jump List")
				assert.NotNil(t, row.DestListEntry, "expected non-nil DestListEntry when parsing Automatic Jump List")
			}
		}
	}
}

func TestGetUserProfiles(t *testing.T) {
	log := logger.New(os.Stdout, true)
	userProfiles, err := getUserProfiles(log, &MockClient{t: t})
	assert.NoError(t, err, "expected no error when getting user profiles")
	assert.NotEmpty(t, userProfiles, "expected non-empty user profiles")
}

func TestGetJumplists(t *testing.T) {
	log := logger.New(os.Stdout, true)
	userProfiles, err := getUserProfiles(log, &MockClient{t: t})
	assert.NoError(t, err, "expected no error when getting user profiles")
	for _, userProfile := range userProfiles {
		jumplists := userProfile.getJumplists(log)
		for _, jumplist := range jumplists {
			log.Infof("found jumplist: %s, username: %s, sid: %s", jumplist.Path, userProfile.Username, userProfile.Sid)
		}
	}
}
