// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"fmt"
	"io/ioutil"

	"github.com/otiai10/copy"
)

type LocalSource struct {
	OrigPath    string `config:"path"`
	workingPath string
	BaseSource
}

func (l *LocalSource) Fetch() (err error) {
	l.workingPath, err = ioutil.TempDir("/tmp", "elastic-synthetics-")
	if err != nil {
		return fmt.Errorf("could not create tmp dir: %w", err)
	}

	err = copy.Copy(l.OrigPath, l.workingPath)
	if err != nil {
		return fmt.Errorf("could not copy suite: %w", err)
	}
	return nil
}

func (l *LocalSource) Workdir() string {
	return l.workingPath
}
