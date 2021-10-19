// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hostfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	defaultMount      = "/hostfs"
	envHostFSOverride = "ELASTIC_OSQUERY_HOSTFS" // Allows to override the mount point for hostfs, default is /hostfs
)

var (
	ErrMissingField     = errors.New("missing/invalid field")
	ErrInvalidFieldType = errors.New("invalid field type")
)

type ColumnType int

const (
	ColumnTypeString ColumnType = iota
	ColumnTypeInt
	ColumnTypeUint
)

func (c ColumnType) String() string {
	return [...]string{"string", "int64", "uint64"}[c]
}

type ColumnInfo struct {
	IndexFrom int
	Name      string
	Type      ColumnType
	Optional  bool
}

func GetPath(fp string) string {
	// Check the environment variable for override, otherwise use /hostfs as the mount root
	mountRoot := os.Getenv(envHostFSOverride)
	if mountRoot == "" {
		mountRoot = defaultMount
	}
	return filepath.Join(mountRoot, fp)
}

type StringMap map[string]string

func (m StringMap) Set(fields []string, col ColumnInfo) error {
	if col.IndexFrom >= len(fields) {
		if !col.Optional {
			return fmt.Errorf("failed to read field at index: %d, when total number of fields is: %d, err: %w", col.IndexFrom, len(fields), ErrMissingField)
		}
		m[col.Name] = ""
		return nil
	}

	var err error

	sval := fields[col.IndexFrom]
	// Check that it is convertable to int type
	switch col.Type {
	case ColumnTypeUint:
		// For unsigned values (Apple) the number is parsed as signed int32 then converted to unsigned.
		// This is consistent with osquery `users` table data on Mac OS.
		// osquery> select * from users;
		// +------------+------------+------------+------------+------------------------+-------------------------------------------------+-------------------------------+------------------+--------------------------------------+-----------+
		// | uid        | gid        | uid_signed | gid_signed | username               | description                                     | directory                     | shell            | uuid                                 | is_hidden |
		// +------------+------------+------------+------------+------------------------+-------------------------------------------------+-------------------------------+------------------+--------------------------------------+-----------+
		// | 229        | 4294967294 | 229        | -2         | _avbdeviced            | Ethernet AVB Device Daemon                      | /var/empty                    | /usr/bin/false   | FFFFEEEE-DDDD-CCCC-BBBB-AAAA000000E5 | 0         |
		v, err := strconv.ParseInt(sval, 10, 32)
		if err == nil {
			n := uint32(v)
			sval = strconv.FormatUint(uint64(n), 10)
		}
	case ColumnTypeInt:
		_, err = strconv.ParseInt(sval, 10, 64)
	}

	if err != nil {
		return fmt.Errorf("invalid field type at index: %d, expected %s, err: %w", col.IndexFrom, col.Type.String(), ErrInvalidFieldType)
	}

	m[col.Name] = sval
	return nil
}
