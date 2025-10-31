// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"github.com/forensicanalysis/fslib"
	"github.com/forensicanalysis/fslib/systemfs"
	"www.velocidex.com/golang/regparser"
)

func LoadRegistry(filePath string) (*regparser.Registry, error) {
	// ensure a path was provided
	if filePath == "" {
		return nil, fmt.Errorf("hive file path is empty")
	}

	// try reading the file directly first, which is faster if it works
	content, err := os.ReadFile(filePath)
	if err == nil {
		registry, err := regparser.NewRegistry(bytes.NewReader(content))
		if err == nil {
			return registry, nil
		}
	}

	// fallback to a low level read using fslib
	sourceFS, err := systemfs.New()
	if err != nil {
		return nil, err
	}
	fsPath, err := fslib.ToFSPath(filePath)
	if err != nil {
		return nil, err
	}
	content, err = fs.ReadFile(sourceFS, fsPath)
	if err != nil {
		return nil, err
	}

	registry, err := regparser.NewRegistry(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	return registry, nil
}
