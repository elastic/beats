// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/v8/libbeat/paths"
)

func TestMain(m *testing.M) {
	// Override global beats data dir to avoid creating directories in the working copy.
	tmpdir, err := ioutil.TempDir("", "beats-data-dir")
	if err != nil {
		fmt.Printf("Failed to create temporal data directory: %v\n", err)
		os.Exit(1)
	}
	paths.Paths.Data = tmpdir

	result := m.Run()
	os.RemoveAll(tmpdir)

	os.Exit(result)
}
