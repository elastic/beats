// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package utilities

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"strings"
	"os"
	"github.com/forensicanalysis/fslib"
	"github.com/forensicanalysis/fslib/systemfs"
	"www.velocidex.com/golang/regparser"
)

// MakeTrimmedString trims null characters from the end of a regparser ValueData string
// and returns it as a regular Go string. If the ValueData is a number, it returns its string representation.
// Otherwise, it returns a hex representation of the raw data.
func MakeTrimmedString(vd *regparser.ValueData) string {
	// regparser strings are null-terminated, and will show the trailing
	// nulls when marshalled to JSON. Trim them here.
	// Also, warn if the type is unexpected.

	if vd == nil {
		return ""
	}

	switch vd.Type {
	case regparser.REG_SZ, regparser.REG_EXPAND_SZ:
		return strings.TrimRight(vd.String, "\x00")
	case regparser.REG_DWORD, regparser.REG_QWORD:
		return fmt.Sprintf("%d", vd.Uint64)
	default:
		log.Printf("Warning: unexpected type for ValueData: %d", vd.Type)
		return fmt.Sprintf("%x", vd.Data)
	}
}

// HiveReader reads a registry hive file and provides access to its regparser.Registry.
// implements the HiveReader interface.
type HiveReader struct {
	FilePath         string
}

// Registry reads the registry hive file and returns a regparser.Registry object.
func (hr *HiveReader) Registry() (*regparser.Registry, error) {
	// TODO: Determine if we should be caching the read between calls

	// ensure a path was provided
	if hr.FilePath == "" {
		return nil, fmt.Errorf("hive file path is empty")
	}

	// try reading the file directly first, which is faster if it works
	content, err := os.ReadFile(hr.FilePath)
	if err == nil {
		registry, err := regparser.NewRegistry(bytes.NewReader(content))
		if err == nil {
			return registry, nil
		}
	}

	// fallback to a low level read using fslib
	sourceFS, err := systemfs.New(); if err != nil {
		return nil, err
	}
	fsPath, err := fslib.ToFSPath(hr.FilePath); if err != nil {
		return nil, err
	}
	content, err = fs.ReadFile(sourceFS, fsPath); if err != nil {
		return nil, err
	}

	registry, err := regparser.NewRegistry(bytes.NewReader(content)); if err != nil {
		return nil, err
	}

	return registry, nil
}
