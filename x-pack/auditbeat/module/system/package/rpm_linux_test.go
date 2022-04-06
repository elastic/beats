// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && cgo
// +build linux,cgo

package pkg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRPMPackages(t *testing.T) {
	_, err := os.Stat(rpmPath)
	if os.IsNotExist(err) {
		t.Skipf("RPM test only on systems where %v exists", rpmPath)
	} else if err != nil {
		t.Fatal(err)
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
