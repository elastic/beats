// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hostfs

import (
	"bufio"
	"os"
	"strings"
)

var columns = []ColumnInfo{
	{0, "groupname", ColumnTypeString, false},
	{2, "gid", ColumnTypeUint, false},
	{2, "gid_signed", ColumnTypeInt, false},
}

func ReadGroup(fn string) ([]map[string]string, error) {
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

		for _, col := range columns {
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
