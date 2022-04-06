// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hostfs

import (
	"bufio"
	"os"
	"strings"
)

var passwdColumns = []ColumnInfo{
	{0, "username", ColumnTypeString, false},
	{2, "uid", ColumnTypeUint, false},
	{2, "uid_signed", ColumnTypeInt, false},
	{3, "gid", ColumnTypeUint, false},
	{3, "gid_signed", ColumnTypeInt, false},
	{4, "description", ColumnTypeString, false},
	{5, "directory", ColumnTypeString, false},
	{6, "shell", ColumnTypeString, false},
	{7, "uuid", ColumnTypeString, true},
}

func ReadPasswd(fn string) ([]map[string]string, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var res []map[string]string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")

		rec := make(StringMap)

		for _, col := range passwdColumns {
			err = rec.Set(fields, col)
			if err != nil {
				return nil, err
			}
		}

		res = append(res, rec)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return res, nil
}
