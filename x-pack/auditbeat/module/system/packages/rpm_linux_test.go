// +build linux

package packages

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
