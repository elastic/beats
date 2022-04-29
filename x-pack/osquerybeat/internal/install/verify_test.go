// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

// setupFiles helper function that creates subdirectory with a given set of files
// The verification currently checks for the file presence only
func setupFiles(testdataBaseDir string, files []string) (string, error) {
	testdir, err := ioutil.TempDir(testdataBaseDir, "")
	if err != nil {
		return "", err
	}

	for _, f := range files {
		fp := filepath.Join(testdir, f)

		dir := filepath.Dir(fp)
		err = os.MkdirAll(dir, 0750)
		if err != nil {
			return "", err
		}

		err = ioutil.WriteFile(fp, nil, 0750)
		if err != nil {
			return "", err
		}
	}

	return testdir, nil
}

func TestVerify(t *testing.T) {
	log := logp.NewLogger("verify_test")
	tests := []struct {
		name  string
		goos  string
		files []string
		err   error
	}{
		{
			name: "darwin no files",
			goos: "darwin",
			err:  os.ErrNotExist,
		},
		{
			name: "linux no files",
			goos: "linux",
			err:  os.ErrNotExist,
		},
		{
			name: "windows no files",
			goos: "windows",
			err:  os.ErrNotExist,
		},
		{
			name:  "darwin extension file missing",
			goos:  "darwin",
			files: []string{"osquery.app/Contents/MacOS/osqueryd"},
			err:   os.ErrNotExist,
		},
		{
			name:  "darwin osqueryd missing",
			goos:  "darwin",
			files: []string{"osquery-extension.ext"},
			err:   os.ErrNotExist,
		},
		{
			name:  "darwin valid install",
			goos:  "darwin",
			files: []string{"osquery.app/Contents/MacOS/osqueryd", "osquery-extension.ext"},
		},
		{
			name:  "linux extension file missing",
			goos:  "linux",
			files: []string{"osqueryd"},
			err:   os.ErrNotExist,
		},
		{
			name:  "linux osqueryd missing",
			goos:  "linux",
			files: []string{"osquery-extension.ext"},
			err:   os.ErrNotExist,
		},
		{
			name:  "linux valid install",
			goos:  "linux",
			files: []string{"osqueryd", "osquery-extension.ext"},
		},
		{
			name:  "windows extension file missing",
			goos:  "windows",
			files: []string{"osqueryd.exe"},
			err:   os.ErrNotExist,
		},
		{
			name:  "windows osqueryd missing",
			goos:  "windows",
			files: []string{"osquery-extension.exe"},
			err:   os.ErrNotExist,
		},
		{
			name:  "windows valid install",
			goos:  "windows",
			files: []string{"osqueryd.exe", "osquery-extension.exe"},
		},
	}

	// Setup test data
	testdataBaseDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := setupFiles(testdataBaseDir, tc.files)
			if err != nil {
				t.Fatal(err)
			}

			err = Verify(tc.goos, dir, log)
			// check for matching error if tc exppect error
			if err != nil {
				if tc.err != nil {
					if !errors.Is(err, tc.err) {
						t.Fatalf("want error: %v, got: %v", tc.err, err)
					}
				} else {
					t.Fatalf("want error: nil, got: %v", err)
				}
			} else if tc.err != nil {
				t.Fatalf("want error: %v, got: nil", tc.err)
			}
		})
	}

}
