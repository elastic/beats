// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package pkg

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func rpmPackagesByExec() ([]*Package, error) {
	format := "%{NAME}|%{VERSION}|%{RELEASE}|%{ARCH}|%{LICENSE}|%{INSTALLTIME}|%{SIZE}|%{URL}|%{SUMMARY}\\n"
	out, err := exec.Command("/usr/bin/rpm", "--qf", format, "-qa").Output()
	if err != nil {
		return nil, fmt.Errorf("Error running rpm -qa command: %v", err)
	}

	lines := strings.Split(string(out), "\n")
	var packages []*Package
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		words := strings.SplitN(line, "|", 9)
		if len(words) < 9 {
			return nil, fmt.Errorf("line '%s' doesn't have enough elements", line)
		}
		pkg := Package{
			Name:    words[0],
			Version: words[1],
			Release: words[2],
			Arch:    words[3],
			License: words[4],
			// install time - 5
			// size - 6
			URL:     words[7],
			Summary: words[8],
			Type:    "rpm",
		}
		ts, err := strconv.ParseInt(words[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error converting %s to string: %v", words[5], err)
		}
		pkg.InstallTime = time.Unix(ts, 0)

		pkg.Size, err = strconv.ParseUint(words[6], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error converting %s to string: %v", words[6], err)
		}

		// Avoid "(none)" in favor of empty strings
		if pkg.URL == "(none)" {
			pkg.URL = ""
		}
		if pkg.Arch == "(none)" {
			pkg.Arch = ""
		}

		packages = append(packages, &pkg)

	}

	return packages, nil
}
