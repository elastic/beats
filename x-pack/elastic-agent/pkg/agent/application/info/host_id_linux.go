// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"os"
	"os/exec"

	"github.com/hashicorp/go-multierror"
)

func loadHostID() (string, error) {
	var mErr error
	p, err := os.ReadFile("/var/lib/dbus/machine-id")
	if err != nil {
		mErr = multierror.Append(mErr, err)
		p, err = os.ReadFile("/etc/machine-id")
		if err != nil {
			mErr = multierror.Append(mErr, err)
			c := exec.Command("hostid") // used for rhel/centos6
			p, err = c.Output()
			if err != nil {
				return "", multierror.Append(mErr, err)
			}
		}
	}
	return string(p), nil
}
