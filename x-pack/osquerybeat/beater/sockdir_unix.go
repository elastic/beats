// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package beater

import (
	"io/ioutil"
	"os"
	"syscall"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func createSockDir(log *logp.Logger) (string, func(), error) {
	// Try to create socket in /var/run first
	// This would result in something the directory something like: /var/run/027202467
	tpath, err := ioutil.TempDir("/var/run", "")
	if err != nil {
		if perr, ok := err.(*os.PathError); ok {
			if perr.Err == syscall.EACCES {
				log.Warnf("Failed to access the directory %s, running as non-root?", perr.Path)
				tpath, err = ioutil.TempDir("", "")
				if err != nil {
					return "", nil, err
				}
			}
		}
	}

	return tpath, func() {
		os.RemoveAll(tpath)
	}, nil
}
