// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"fmt"
	"os/exec"
	"regexp"
)

var winReg = regexp.MustCompile("[\\s]+MachineGuid[\\s]+REG_SZ[\\s]+([\\S]+)")

func loadHostID() (string, error) {
	c := exec.Command("Reg", "query", "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Cryptography", "/v", "MachineGuid")
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	matches := winReg.FindSubmatch(out)
	if len(matches) < 2 {
		return "", fmt.Errorf("unable to find MachineGuid in output %q", out)
	}
	return string(matches[1]), nil
}
