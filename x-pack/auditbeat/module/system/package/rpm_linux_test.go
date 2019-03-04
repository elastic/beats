// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,cgo

package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRPMPackages(t *testing.T) {
	os, err := getOS()
	if err != nil {
		t.Fatal(err)
	}

	if os.Family != "redhat" {
		t.Skip("RPM test only on Redhat systems")
	}

	// Control using the exec command
	packagesExpected, err := rpmPackagesByExec()
	if err != nil {
		t.Fatal(err)
	}

	packages, err := listRPMPackages()
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, packagesExpected, packages)

}
