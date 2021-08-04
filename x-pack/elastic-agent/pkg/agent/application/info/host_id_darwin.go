// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"fmt"
	"os/exec"
	"regexp"
)

var macReg = regexp.MustCompile(`[\s]+"IOPlatformUUID" = "([\S]+)"`)

func loadHostID() (string, error) {
	c := exec.Command("Ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	matches := macReg.FindSubmatch(out)
	if len(matches) < 2 {
		return "", fmt.Errorf("unable to find IOPLatformUUID in %q", out)
	}
	return string(matches[1]), nil
}
