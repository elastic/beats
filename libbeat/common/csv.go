// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package common

import (
	"bytes"
	"encoding/csv"
	"strings"
)

// DumpInCSVFormat takes a set of fields and rows and returns a string
// representing the CSV representation for the fields and rows.
func DumpInCSVFormat(fields []string, rows [][]string) string {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	for i, field := range fields {
		fields[i] = strings.Replace(field, "\n", "\\n", -1)
	}
	if len(fields) > 0 {
		writer.Write(fields)
	}

	for _, row := range rows {
		for i, field := range row {
			field = strings.Replace(field, "\n", "\\n", -1)
			field = strings.Replace(field, "\r", "\\r", -1)
			row[i] = field
		}
		writer.Write(row)
	}
	writer.Flush()

	csv := buf.String()
	return csv
}
