// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build freebsd netbsd openbsd

package info

import (
	"os"
	"os/exec"

	"github.com/hashicorp/go-multierror"
)

func loadHostID() (string, error) {
	var mErr error
	p, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		mErr = multierror.Append(mErr, err)
		c := exec.Command("kenv", "-q", "smbios.system.uuid")
		p, err = c.Output()
		if err != nil {
			return "", multierror.Append(mErr, err)
		}
	}
	return string(p), nil
}
