// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package utilities

import (
	"bytes"
	"fmt"
	"github.com/forensicanalysis/fslib"
	"github.com/forensicanalysis/fslib/systemfs"
	"github.com/osquery/osquery-go/plugin/table"
	"io/fs"
	"log"
	"os"
	"strings"
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

// Registry reads the registry hive file and returns a regparser.Registry object.
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

func GetConstraintsFromQueryContext(fieldName string, context table.QueryContext) []string {
	constraints := make([]string, 0)
	for name, cList := range context.Constraints {
		if len(cList.Constraints) > 0 && name == fieldName {
			for _, c := range cList.Constraints {
				log.Printf("%s Query constraint: %d %s", fieldName, c.Operator, c.Expression)
				if c.Operator != table.OperatorEquals {
					log.Printf("Warning: only '=' operator is supported for %s constraints, skipping %d", fieldName, c.Operator)
					continue
				}
				constraints = append(constraints, c.Expression)
			}
		}
	}
	return constraints
}
